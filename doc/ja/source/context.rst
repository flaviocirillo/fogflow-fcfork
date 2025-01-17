*****************************************************
分散コンテキスト管理 (Distributed context management)
*****************************************************

コンテキスト管理システムは、すべてのシステム コンポーネントと実行中のタスク インスタンスにグローバル ビューを提供して、統合データモデルと通信プロトコル NGSI を介してコンテキスト エンティティをクエリ、サブスクライブ、およびアップデートするように設計されています。
これは、FogFlow で標準ベースのエッジ プログラミング モデルをサポートするために非常に重要な役割を果たします。MQTT ベースの Mosquitto や Apache Kafka などの他の既存のブローカーと比較すると、FogFlow の分散コンテキスト管理システムには次の機能があります:

* コンテキストの可用性とコンテキスト エンティティを分離します。
* コンテキスト データ (NGSI10 経由) とコンテキスト可用性 (NGSI9 経由) の両方を管理するための分離された標準化されたインターフェイスを提供します。
* ID ベースおよびトピック ベースのクエリとサブスクリプションだけでなく、ジオスコープ ベースのクエリとサブスクリプションもサポートします

次の図に示すように、FogFlow では、グローバルで集中化された IoT Discovery の調整の下で、多数の分散型 IoT Broker が並行して動作します。

.. figure:: ../../en/source/figures/distributed-brokers.png



IoT Discovery
===============================
一元化された IoT Discovery は、コンテキスト データのコンテキスト可用性のグローバルビューを提供し、コンテキスト可用性のレジストレーション、ディスカバリー、およびサブスクリプションのための NGSI9 インターフェイスを提供します。

IoT Broker
===============================
FogFlow の IoTBroker は、各コンテキスト エンティティの最新の値のみを保持し、各エンティティ データをシステムメモリに直接保存するため、非常に軽量です。これにより、コンテキスト プロデューサーからコンテキスト コンシューマーへのデータ転送に高スループットと低レイテンシーがもたらされます。

各 IoT Broker は、コンテキスト データの一部を管理し、データを共有 IoT Discovery に登録します。ただし、すべての IoT Broker は、共有 IoT Discovery を介してどの IoT Broker がエンティティを提供するかを見つけて、そのリモート IoT Broker からエンティティをフェッチできるため、NGSI10 を介して要求されたコンテキスト エンティティを等しく提供できます。
