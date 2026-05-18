const amqplib = require('amqplib');

const exchange_name = 'fogflow';
const exchange_type = 'topic';
const queue_name = 'fogflow-designer';
const subscribed_keys = ['designer.*', 'task.'];
const TIME_INTERVAL_RECONNECT = 5000;

let amqp_url = null;
let amqpConn = null;
let amqpChannel = null;
let msgHandler = null;
let cb_after_connected = null;

let isReady = false;
let isConnecting = false;
let reconnectTimer = null;
let hasCalledAfterConnected = false;
const pendingMessages = [];

function Init(rabbitmqURL, fnConsumer, afterConnected) {
    amqp_url = rabbitmqURL;
    msgHandler = fnConsumer;
    cb_after_connected = afterConnected;
    connect();
}

function scheduleReconnect() {
    if (reconnectTimer) {
        return;
    }
    reconnectTimer = setTimeout(() => {
        reconnectTimer = null;
        connect();
    }, TIME_INTERVAL_RECONNECT);
}

function onDisconnected(reason) {
    if (!isReady && !amqpConn && !amqpChannel) {
        scheduleReconnect();
        return;
    }

    console.error('[RabbitMQ] disconnected:', reason || 'unknown');
    isReady = false;
    isConnecting = false;
    amqpChannel = null;

    if (amqpConn) {
        try {
            amqpConn.removeAllListeners();
        } catch (_) { /* ignore */ }
        amqpConn = null;
    }

    scheduleReconnect();
}

async function connect() {
    if (isConnecting) {
        return;
    }
    isConnecting = true;
    console.log('[RabbitMQ] connecting to', amqp_url);

    try {
        const conn = await amqplib.connect(amqp_url);
        amqpConn = conn;

        conn.on('error', (err) => {
            console.error('[RabbitMQ] connection error:', err.message);
        });
        conn.on('close', () => {
            onDisconnected('connection closed');
        });

        const channel = await conn.createChannel();
        amqpChannel = channel;

        channel.on('error', (err) => {
            console.error('[RabbitMQ] channel error:', err.message);
        });
        channel.on('close', () => {
            if (isReady) {
                onDisconnected('channel closed');
            }
        });

        await channel.assertExchange(exchange_name, exchange_type, {
            durable: true,
            autoDelete: true,
        });
        await channel.assertQueue(queue_name, { durable: true });

        for (let i = 0; i < subscribed_keys.length; i++) {
            const key = subscribed_keys[i];
            console.log('[RabbitMQ] subscribed to', key);
            await channel.bindQueue(queue_name, exchange_name, key);
        }

        await channel.consume(queue_name, processMsg, { noAck: true });

        isReady = true;
        isConnecting = false;
        console.log('[RabbitMQ] connected');

        if (!hasCalledAfterConnected && cb_after_connected) {
            hasCalledAfterConnected = true;
            cb_after_connected();
        }

        await flushPendingMessages();
    } catch (err) {
        isConnecting = false;
        console.error('[RabbitMQ]', err.message);
        amqpChannel = null;
        if (amqpConn) {
            try {
                amqpConn.removeAllListeners();
            } catch (_) { /* ignore */ }
            amqpConn = null;
        }
        scheduleReconnect();
    }
}

function processMsg(msg) {
    const jsonMsg = JSON.parse(msg.content);
    msgHandler(jsonMsg);
}

async function publishNow(msg) {
    if (!isReady || !amqpChannel) {
        throw new Error('channel not ready');
    }

    const msgContent = JSON.stringify(msg);
    const ok = amqpChannel.publish(
        exchange_name,
        msg.RoutingKey,
        Buffer.from(msgContent),
        { contentType: 'application/json', persistent: true }
    );

    if (!ok) {
        await new Promise((resolve) => amqpChannel.once('drain', resolve));
    }
}

async function flushPendingMessages() {
    while (pendingMessages.length > 0 && isReady && amqpChannel) {
        const item = pendingMessages[0];
        try {
            await publishNow(item.msg);
            pendingMessages.shift();
            item.resolve();
        } catch (err) {
            console.error('[RabbitMQ] failed to publish queued message:', err.message);
            if (!isReady || !amqpChannel) {
                break;
            }
            pendingMessages.shift();
            item.reject(err);
        }
    }
}

function Publish(msg) {
    return new Promise((resolve, reject) => {
        const item = { msg, resolve, reject };

        if (isReady && amqpChannel) {
            publishNow(msg)
                .then(resolve)
                .catch((err) => {
                    console.error('[RabbitMQ] publish failed, re-queuing:', err.message);
                    pendingMessages.push(item);
                    onDisconnected(err.message);
                });
            return;
        }

        console.log(
            '[RabbitMQ] channel not ready, queuing message (pending:',
            pendingMessages.length + 1,
            ')'
        );
        pendingMessages.push(item);

        if (!isConnecting && !reconnectTimer) {
            connect();
        }
    });
}

module.exports = { Init, Publish };
