package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/google/uuid"
	_ "github.com/lib/pq"

	. "fogflow/common/config"
	. "fogflow/common/ngsi"
)

type InterSiteSubscription struct {
	DiscoveryURL   string
	SubscriptionID string
}

type CacheItem struct {
	SubscriberURL string
	Notify        *NotifyContextAvailabilityRequest
}

type FastDiscovery struct {
	//backend entity repository
	repository EntityRepository

	cfg         *Config
	SecurityCfg *HTTPS

	//list of active brokers within the same site
	BrokerList       map[string]*BrokerProfile
	broker_list_lock sync.RWMutex

	//mapping from subscriptionID to subscription
	subscriptions      map[string]*SubscribeContextAvailabilityRequest
	subscriptions_lock sync.RWMutex

	//cache
	notifyCache []*CacheItem
	cache_lock  sync.RWMutex

	// Configure store on disk
	delayStoreOnFile int
	storeOnDisk      bool

	// lock to control the update subscriptions in database
	subscriptionsDbLock               sync.RWMutex
	storeSubscriptionsOnFileScheduled bool

	// lock to control the update brokers in database
	brokersDbLock               sync.RWMutex
	storeBrokersOnFileScheduled bool

	dbFiles map[string]string
}

func (fd *FastDiscovery) Init(config *Config) {
	fd.subscriptions = make(map[string]*SubscribeContextAvailabilityRequest)
	fd.BrokerList = make(map[string]*BrokerProfile)
	fd.notifyCache = make([]*CacheItem, 0)

	fd.cfg = config
	fd.SecurityCfg = &config.HTTPS

	fd.storeSubscriptionsOnFileScheduled = false
	fd.storeBrokersOnFileScheduled = false
	fd.storeOnDisk = config.Discovery.StoreOnDisk
	//INFO.Println("config.Discovery.DelayStoreRegistrationsOnFile ", config.Discovery.DelayStoreStoreOnFile)
	fd.delayStoreOnFile = config.Discovery.DelayStoreOnFile

	fd.dbFiles = make(map[string]string)
	fd.dbFiles["brokers"] = "discoveryDB/brokers.json"
	fd.dbFiles["subscriptions"] = "discoveryDB/subscriptions.json"
	fd.dbFiles["registrations"] = "discoveryDB/registrations.json"

	if fd.storeOnDisk {
		fd.readSubscriptionsFromDisk()
		fd.readBrokersFromDisk()
	}

	fd.repository.Init(config)
}

func (fd *FastDiscovery) Stop() {

}

func (fd *FastDiscovery) RegisterContext(w rest.ResponseWriter, r *rest.Request) {
	registerCtxReq := RegisterContextRequest{}
	err := r.DecodeJsonPayload(&registerCtxReq)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if registerCtxReq.RegistrationId == "" {
		u1, err := uuid.NewUUID()
		if err != nil {
			rest.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		registrationID := u1.String()
		registerCtxReq.RegistrationId = registrationID
	}
	// update context registration
	go fd.updateRegistration(&registerCtxReq)

	// send out the response
	registerCtxResp := RegisterContextResponse{}
	registerCtxResp.RegistrationId = registerCtxReq.RegistrationId
	registerCtxResp.Duration = registerCtxReq.Duration
	registerCtxResp.ErrorCode.Code = 200
	registerCtxResp.ErrorCode.ReasonPhrase = "OK"
	w.WriteJson(&registerCtxResp)
}

func (fd *FastDiscovery) forwardRegistrationCtxAvailability(discoveryURL string, registrationReq *RegisterContextRequest) {
	requestURL := "http://" + discoveryURL + "/ngsi9/registerContext"
	client := NGSI9Client{IoTDiscoveryURL: requestURL}
	_, err := client.RegisterContext(registrationReq)
	if err != nil {
		ERROR.Println(err)
	}
}

func (fd *FastDiscovery) notifySubscribers(registration *EntityRegistration, updateAction string) {
	fd.subscriptions_lock.RLock()
	defer fd.subscriptions_lock.RUnlock()
	providerURL := registration.ProvidingApplication
	for _, subscription := range fd.subscriptions {
		// find out the updated entities matched with this subscription
		if matchingWithFilters(registration, subscription.Entities, subscription.Attributes, subscription.Restriction) == true {
			subscriberURL := subscription.Reference
			subID := subscription.SubscriptionId
			entities := make([]EntityId, 0)

			entity := EntityId{}
			entity.ID = registration.ID
			entity.Type = registration.Type
			entity.IsPattern = false
			entities = append(entities, entity)

			entityMap := make(map[string][]EntityId)
			entityMap[providerURL] = entities

			// send out AvailabilityNotify to subscribers
			go fd.sendNotify(subID, subscriberURL, entityMap, updateAction)
		}
	}
}

func (fd *FastDiscovery) updateRegistration(registReq *RegisterContextRequest) {
	for _, registration := range registReq.ContextRegistrations {
		for _, entity := range registration.EntityIdList {
			// update the registration, both in the memory cache and in the database
			updatedRegistration := fd.repository.updateEntity(entity, &registration)

			// inform the associated subscribers after updating the repository
			go fd.notifySubscribers(updatedRegistration, "UPDATE")
		}
	}
}

func (fd *FastDiscovery) deleteRegistration(eid string) {
	registration := fd.repository.retrieveRegistration(eid)
	if registration == nil {
		INFO.Println("the entity registration does not exist: ", eid)
		return
	}

	fd.notifySubscribers(registration, "DELETE")

	// if the registration is for the broker itself, we need to update the broker list
	if registration.Type == "IoTBroker" {
		brokerID := registration.ID
		fd.broker_list_lock.Lock()
		delete(fd.BrokerList, brokerID)
		fd.broker_list_lock.Unlock()

		if fd.storeOnDisk {
			go fd.updateBrokersOnDisk()
		}
	}

	fd.repository.deleteEntity(eid)
}

func (fd *FastDiscovery) DiscoverContextAvailability(w rest.ResponseWriter, r *rest.Request) {
	discoverCtxReq := DiscoverContextAvailabilityRequest{}
	err := r.DecodeJsonPayload(&discoverCtxReq)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// query all relevant discovery instances to get the matched result
	result := make([]ContextRegistrationResponse, 0)

	registrationList := fd.handleQueryCtxAvailability(&discoverCtxReq)
	for _, registration := range registrationList {
		result = append(result, registration)
	}

	// send out the response
	discoverCtxResp := DiscoverContextAvailabilityResponse{}
	if result == nil {
		discoverCtxResp.ErrorCode.Code = 500
		discoverCtxResp.ErrorCode.ReasonPhrase = "database is too overloaded"
	} else {
		discoverCtxResp.ContextRegistrationResponses = result
		discoverCtxResp.ErrorCode.Code = 200
		discoverCtxResp.ErrorCode.ReasonPhrase = "OK"
	}
	w.WriteJson(&discoverCtxResp)
}

func (fd *FastDiscovery) handleQueryCtxAvailability(req *DiscoverContextAvailabilityRequest) []ContextRegistrationResponse {
	entityMap := fd.repository.queryEntities(req.Entities, req.Attributes, req.Restriction)

	// prepare the response
	registrationList := make([]ContextRegistrationResponse, 0)

	for url, entity := range entityMap {
		resp := ContextRegistrationResponse{}
		resp.ContextRegistration.ProvidingApplication = url
		resp.ContextRegistration.EntityIdList = entity

		resp.ErrorCode.Code = 200
		resp.ErrorCode.ReasonPhrase = "OK"

		registrationList = append(registrationList, resp)
	}

	return registrationList
}

func (fd *FastDiscovery) SubscribeContextAvailability(w rest.ResponseWriter, r *rest.Request) {
	subscribeCtxAvailabilityReq := SubscribeContextAvailabilityRequest{}
	err := r.DecodeJsonPayload(&subscribeCtxAvailabilityReq)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// generate a new subscription id
	u1, err := uuid.NewUUID()
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	subID := u1.String()

	subscribeCtxAvailabilityReq.SubscriptionId = subID

	// add the new subscription
	fd.subscriptions_lock.Lock()
	fd.subscriptions[subID] = &subscribeCtxAvailabilityReq

	fd.subscriptions_lock.Unlock()

	if fd.storeOnDisk {
		go fd.updateSubscriptionsOnDisk()
	}

	// send out the response
	subscribeCtxAvailabilityResp := SubscribeContextAvailabilityResponse{}
	subscribeCtxAvailabilityResp.SubscriptionId = subID
	subscribeCtxAvailabilityResp.Duration = subscribeCtxAvailabilityReq.Duration
	subscribeCtxAvailabilityResp.ErrorCode.Code = 200
	subscribeCtxAvailabilityResp.ErrorCode.ReasonPhrase = "OK"

	w.WriteJson(&subscribeCtxAvailabilityResp)
	// trigger the process to send out the matched context availability infomation to the subscriber
	go fd.handleSubscribeCtxAvailability(&subscribeCtxAvailabilityReq)
}

//receive updateContextAvailability for subscription
func (fd *FastDiscovery) UpdateLDContextAvailability(w rest.ResponseWriter, r *rest.Request) {
	sid := r.PathParam("sid")
	subscribeCtxAvailabilityReq := SubscribeContextAvailabilityRequest{}
	err := r.DecodeJsonPayload(&subscribeCtxAvailabilityReq)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	subID := sid
	subscribeCtxAvailabilityReq.SubscriptionId = subID

	// add the new subscription
	fd.subscriptions_lock.Lock()
	if sid != "" {
		fd.subscriptions[subID] = &subscribeCtxAvailabilityReq
	}
	fd.subscriptions_lock.Unlock()

	if fd.storeOnDisk {
		go fd.updateSubscriptionsOnDisk()
	}

	// send out the response
	subscribeCtxAvailabilityResp := SubscribeContextAvailabilityResponse{}
	subscribeCtxAvailabilityResp.SubscriptionId = subID
	subscribeCtxAvailabilityResp.Duration = subscribeCtxAvailabilityReq.Duration
	subscribeCtxAvailabilityResp.ErrorCode.Code = 200
	subscribeCtxAvailabilityResp.ErrorCode.ReasonPhrase = "OK"

	w.WriteJson(&subscribeCtxAvailabilityResp)
	// trigger the process to send out the matched context availability infomation to the subscriber
	if sid != "" {
		go fd.handleSubscribeCtxAvailability(&subscribeCtxAvailabilityReq)
	}
}

// handle NGSI9 subscription based on the local database
func (fd *FastDiscovery) handleSubscribeCtxAvailability(subReq *SubscribeContextAvailabilityRequest) {
	entityMap := fd.repository.queryEntities(subReq.Entities, subReq.Attributes, subReq.Restriction)
	if len(entityMap) > 0 {
		fd.sendNotify(subReq.SubscriptionId, subReq.Reference, entityMap, "CREATE")
	}
}

func (fd *FastDiscovery) sendNotify(subID string, subscriberURL string, entityMap map[string][]EntityId, action string) {
	notifyReq := NotifyContextAvailabilityRequest{}
	notifyReq.SubscriptionId = subID

	// carry the actions via the code number
	switch action {
	case "CREATE":
		notifyReq.ErrorCode.Code = 201
	case "UPDATE":
		notifyReq.ErrorCode.Code = 301
	case "DELETE":
		notifyReq.ErrorCode.Code = 410
	}

	notifyReq.ErrorCode.ReasonPhrase = "OK"

	// prepare the response
	registrationList := make([]ContextRegistrationResponse, 0)

	for url, entity := range entityMap {
		resp := ContextRegistrationResponse{}
		resp.ContextRegistration.ProvidingApplication = url
		resp.ContextRegistration.EntityIdList = entity

		resp.ErrorCode.Code = 200

		resp.ErrorCode.ReasonPhrase = "OK"

		registrationList = append(registrationList, resp)
	}

	notifyReq.ContextRegistrationResponseList = registrationList

	// if there are some notificaitons already in the tmpCache, send those in the cache first
	fd.cache_lock.RLock()
	if len(fd.notifyCache) > 0 {
		//try to send the items in the cache
		go fd.resendCachedItems()
	}
	fd.cache_lock.RUnlock()

	//send the current notify
	done := fd.postNotify(subscriberURL, &notifyReq)
	if done == false { // put it into the tmpCache
		fd.cache_lock.Lock()
		item := CacheItem{}
		item.SubscriberURL = subscriberURL
		item.Notify = &notifyReq
		fd.notifyCache = append(fd.notifyCache, &item)
		fd.cache_lock.Unlock()
	}
}

func (fd *FastDiscovery) resendCachedItems() {
	fd.cache_lock.Lock()
	cachedItem := fd.notifyCache
	fd.notifyCache = make([]*CacheItem, 0)
	fd.cache_lock.Unlock()

	// resend them
	newCache := make([]*CacheItem, 0)

	for _, item := range cachedItem {
		err := fd.postNotify(item.SubscriberURL, item.Notify)
		if err == false {
			newCache = append(newCache, item)
		}
	}

	// if some of them are still failed, put them back into the cache
	if len(newCache) > 0 {
		fd.cache_lock.Lock()
		fd.notifyCache = append(fd.notifyCache, newCache...)
		fd.cache_lock.Unlock()
	}
}

func (fd *FastDiscovery) OnTimer() {
	// try to send the items in the cache
	fd.resendCachedItems()

	// check the liveness of brokers
	fd.checkBrokerList()
}

func (fd *FastDiscovery) postNotify(subscriberURL string, notifyReq *NotifyContextAvailabilityRequest) bool {
	body, err := json.Marshal(notifyReq)
	if err != nil {
		ERROR.Println(err)
		return false
	}
	req, err := http.NewRequest("POST", subscriberURL, bytes.NewBuffer(body))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	client := &http.Client{}
	if strings.HasPrefix(subscriberURL, "https") == true {
		client = fd.SecurityCfg.GetHTTPClient()
	}

	resp, err2 := client.Do(req)
	if err2 != nil {
		ERROR.Println(err2)
		return false
	}
	defer resp.Body.Close()

	text, _ := ioutil.ReadAll(resp.Body)

	notifyCtxAvailResp := NotifyContextAvailabilityResponse{}
	err = json.Unmarshal(text, &notifyCtxAvailResp)
	if err != nil {
		ERROR.Println(err)
		return false
	}

	if notifyCtxAvailResp.ResponseCode.Code != 200 {
		ERROR.Println(notifyCtxAvailResp.ResponseCode.ReasonPhrase)
		return false
	}

	return true
}

func (fd *FastDiscovery) UnsubscribeContextAvailability(w rest.ResponseWriter, r *rest.Request) {
	unsubscribeCtxAvailabilityReq := UnsubscribeContextAvailabilityRequest{}
	err := r.DecodeJsonPayload(&unsubscribeCtxAvailabilityReq)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	subID := unsubscribeCtxAvailabilityReq.SubscriptionId

	// remove the subscription
	fd.subscriptions_lock.Lock()
	delete(fd.subscriptions, subID)
	fd.subscriptions_lock.Unlock()

	if fd.storeOnDisk {
		go fd.updateSubscriptionsOnDisk()
	}

	// send out the response
	unsubscribeCtxAvailabilityResp := UnsubscribeContextAvailabilityResponse{}
	unsubscribeCtxAvailabilityResp.SubscriptionId = unsubscribeCtxAvailabilityReq.SubscriptionId
	unsubscribeCtxAvailabilityResp.StatusCode.Code = 200
	unsubscribeCtxAvailabilityResp.StatusCode.Details = "OK"

	w.WriteJson(&unsubscribeCtxAvailabilityResp)
}

func (fd *FastDiscovery) getRegisteredLDEntity(w rest.ResponseWriter, r *rest.Request) {
	var eid = r.PathParam("eid")
	var newEid string
	if r.Header.Get("fiware-service") != "" {
		newEid = eid + "@" + r.Header.Get("fiware-service")
		w.Header().Set("fiware-service", r.Header.Get("fiware-service"))
	} else {
		newEid = eid + "@" + "default"
	}
	registration := fd.repository.retrieveRegistration(newEid)
	registration.ID, _ = FiwareId(registration.ID)
	w.WriteJson(registration)
}

func (fd *FastDiscovery) getRegisteredEntity(w rest.ResponseWriter, r *rest.Request) {
	var eid = r.PathParam("eid")
	registration := fd.repository.retrieveRegistration(eid)
	w.WriteJson(registration)
}

func (fd *FastDiscovery) deleteRegisteredEntity(w rest.ResponseWriter, r *rest.Request) {
	var eid = r.PathParam("eid")
	w.WriteHeader(200)

	go fd.deleteRegistration(eid)
}

func (fd *FastDiscovery) getSubscription(w rest.ResponseWriter, r *rest.Request) {
	var sid = r.PathParam("sid")

	fd.subscriptions_lock.RLocker()
	defer fd.subscriptions_lock.RUnlock()

	subscription := fd.subscriptions[sid]
	w.WriteJson(subscription)
}

func (fd *FastDiscovery) getSubscriptions(w rest.ResponseWriter, r *rest.Request) {
	fd.subscriptions_lock.RLock()
	defer fd.subscriptions_lock.RUnlock()

	w.WriteJson(fd.subscriptions)
}

func (fd *FastDiscovery) getEntityTypes(w rest.ResponseWriter, r *rest.Request) {
	w.WriteHeader(200)
	w.WriteJson(fd.repository.GetEntityTypes())
}

func (fd *FastDiscovery) getStatus(w rest.ResponseWriter, r *rest.Request) {
	w.WriteHeader(200)
	w.WriteJson("ok")
}

func (fd *FastDiscovery) getBrokerList(w rest.ResponseWriter, r *rest.Request) {
	fd.broker_list_lock.RLock()
	defer fd.broker_list_lock.RUnlock()

	w.WriteHeader(200)
	w.WriteJson(fd.BrokerList)
}

func (fd *FastDiscovery) onBrokerHeartbeat(w rest.ResponseWriter, r *rest.Request) {
	brokerProfile := BrokerProfile{}

	err := r.DecodeJsonPayload(&brokerProfile)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// send out the response
	updateCtxResp := UpdateContextResponse{}
	w.WriteJson(&updateCtxResp)

	fd.broker_list_lock.Lock()
	defer fd.broker_list_lock.Unlock()

	if broker, exist := fd.BrokerList[brokerProfile.BID]; exist {
		broker.MyURL = brokerProfile.MyURL
		broker.Last_Heartbeat_Update = time.Now()
	} else {
		brokerProfile.Last_Heartbeat_Update = time.Now()
		fd.BrokerList[brokerProfile.BID] = &brokerProfile

		if fd.storeOnDisk {
			go fd.updateBrokersOnDisk()
		}
	}

}

func (fd *FastDiscovery) checkBrokerList() {
	fd.broker_list_lock.Lock()
	defer fd.broker_list_lock.Unlock()

	for brokerID, brokerProfile := range fd.BrokerList {
		if brokerProfile.IsLive(fd.cfg.Broker.HeartbeatInterval*6) == false {
			delete(fd.BrokerList, brokerID)
			INFO.Println("REMOVE broker " + brokerID + " from the list")
			if fd.storeOnDisk {
				go fd.updateBrokersOnDisk()
			}
		}
	}
}

func (fd *FastDiscovery) updateSubscriptionsOnDisk() {

	// This initial code is to avoid to write on file for every new subscriptions if
	// they are arriving too close with each other in terms of time
	// So here we check if the dblock was already taken without trying to take it
	// if it is already taken then just return because the new subscription will be written
	// by the already scheduled write to file
	// If the lock is not taken, then take it and wait for 3 seconds.
	// Then for the next 3 seconds other pursuer of storing on file will see that
	// somebody has the lock

	if fd.storeSubscriptionsOnFileScheduled {
		INFO.Println("A store on file for registrations is already scheduled")
		return
	}

	fd.subscriptionsDbLock.Lock()

	fd.storeSubscriptionsOnFileScheduled = true

	time.Sleep(time.Duration(fd.delayStoreOnFile) * time.Second)

	fd.subscriptions_lock.RLock()

	INFO.Println("Writing subscriptions into file")

	defer fd.subscriptions_lock.RUnlock()
	defer fd.subscriptionsDbLock.Unlock()

	//...................................
	//Writing struct type to a JSON file
	//...................................
	content, err := json.Marshal(fd.subscriptions)
	if err != nil {
		ERROR.Println(err)
	}
	// err = ioutil.WriteFile("subscriptions.json", content, 0644)
	err = ioutil.WriteFile(fd.dbFiles["subscriptions"], content, 0644)
	if err != nil {
		ERROR.Println(err)
	}

	fd.storeSubscriptionsOnFileScheduled = false

}

func (fd *FastDiscovery) readSubscriptionsFromDisk() {

	INFO.Println("Reading subscriptions from file")

	fd.subscriptionsDbLock.Lock()
	defer fd.subscriptionsDbLock.Unlock()

	// content, err := ioutil.ReadFile("subscriptions.json")
	content, err := ioutil.ReadFile(fd.dbFiles["subscriptions"])
	if err != nil {
		ERROR.Println(err)
	}

	err = json.Unmarshal(content, &fd.subscriptions)
	if err != nil {
		ERROR.Println(err)
	}
}

func (fd *FastDiscovery) updateBrokersOnDisk() {

	// This initial code is to avoid to write on file for every new subscriptions if
	// they are arriving too close with each other in terms of time
	// So here we check if the dblock was already taken without trying to take it
	// if it is already taken then just return because the new subscription will be written
	// by the already scheduled write to file
	// If the lock is not taken, then take it and wait for 3 seconds.
	// Then for the next 3 seconds other pursuer of storing on file will see that
	// somebody has the lock

	if fd.storeBrokersOnFileScheduled {
		INFO.Println("A store on file for registrations is already scheduled")
		return
	}

	fd.brokersDbLock.Lock()

	fd.storeBrokersOnFileScheduled = true

	time.Sleep(time.Duration(fd.delayStoreOnFile) * time.Second)

	fd.broker_list_lock.RLock()

	INFO.Println("Writing brokers into file")

	defer fd.broker_list_lock.RUnlock()
	defer fd.brokersDbLock.Unlock()

	//...................................
	//Writing struct type to a JSON file
	//...................................
	content, err := json.Marshal(fd.BrokerList)
	if err != nil {
		ERROR.Println(err)
	}
	// err = ioutil.WriteFile("brokers.json", content, 0644)
	err = ioutil.WriteFile(fd.dbFiles["brokers"], content, 0644)
	if err != nil {
		ERROR.Println(err)
	}

	fd.storeBrokersOnFileScheduled = false

}

func (fd *FastDiscovery) readBrokersFromDisk() {

	INFO.Println("Reading brokers from file")

	fd.brokersDbLock.Lock()
	defer fd.brokersDbLock.Unlock()

	// content, err := ioutil.ReadFile("brokers.json")
	content, err := ioutil.ReadFile(fd.dbFiles["brokers"])
	if err != nil {
		ERROR.Println(err)
	}

	err = json.Unmarshal(content, &fd.BrokerList)
	if err != nil {
		ERROR.Println(err)
	}
}
