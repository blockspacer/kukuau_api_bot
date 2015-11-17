package db

import (
	"crypto/md5"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"time"
	"errors"
	"fmt"
	"io"
	"msngr/structs"

	"reflect"
)

const (
	LOGOUT = "LOGOUT"
	LOGIN = "LOGIN"
	REGISTERED = "REGISTERED"
)

func phash(pwd *string) (*string) {
	input := []byte(*pwd)
	output := md5.Sum(input)
	result := string(output[:])
	return &result
}

func is_index_key_present(currentIndexes []mgo.Index, key []string) bool {
	for _, index := range currentIndexes {
		if reflect.DeepEqual(key, index.Key) {
			return true
		}
	}
	return false
}

type OrderData struct {
	Content map[string]interface{}
}
func NewOrderData(content map[string]interface{}) OrderData {
	return OrderData{Content:content}
}

func (odh *OrderData) Get(key string) interface{} {
	val, ok := odh.Content[key]
	if ok {
		return val
	}else {
		return nil
	}
}

type OrderWrapper struct {
	OrderState int   `bson:"order_state"`
	OrderId    int64 `bson:"order_id"`
	When       time.Time
	Whom       string
	OrderData  OrderData `bson:"data"`
	Feedback   string
	Source     string
}



type UserWrapper struct {
	State      string `bson:"user_state"`
	UserId     *string `bson:"user_id"`
	UserName   *string `bson:"user_name"`
	Password   *string
	Phone      *string

	LastUpdate time.Time `bson:"last_update"`
}

type ErrorWrapper struct {
	Username string
	Error    string
	Time     time.Time
}

type Loaded interface {
	isLoaded() bool
}


type orderHandler struct {
	collection *mgo.Collection
	parent     *DbHandlerMixin
}

type userHandler struct {
	collection *mgo.Collection
	parent     *DbHandlerMixin
}

type errorHandler struct {
	collection *mgo.Collection
	parent     *DbHandlerMixin
}

type DbHandlerMixin struct {
	conn    string
	dbname  string

	session *mgo.Session

	Orders  *orderHandler
	Users   *userHandler
	Errors  *errorHandler
	Check   structs.CheckFunc
}

var DELETE_DB = false

func (odbh *DbHandlerMixin) IsConnected() bool {
	return odbh.session != nil
}

func (odbh *DbHandlerMixin) reConnect() {
	var session *mgo.Session
	count := 2500 * time.Millisecond
	for {
		var err error
		session, err = mgo.Dial(odbh.conn)
		if err == nil {
			log.Printf("Connection to mongodb established!")
			odbh.session = session
			break
		} else {
			count += count
			log.Printf("can not connect to db, will sleep %+v and try", count)
			time.Sleep(count)
		}
	}

	session.SetMode(mgo.Strong, true)
	odbh.session = session

	if (DELETE_DB) {
		log.Printf("will delete database %+v", odbh.dbname)
		err := session.DB(odbh.dbname).DropDatabase()
		if err != nil {
			log.Println("db must be dropped but errr:\n", err)
		}
	}
	orders_collection := session.DB(odbh.dbname).C("orders")
	log.Printf("collection %+v", orders_collection)
	indexes, err := orders_collection.Indexes()
	if err != nil {
		log.Printf("Error at index information: %v", err)
	}
	log.Printf("indexes %+v", indexes)

	if !is_index_key_present(indexes, []string{"order_id"}) {
		orders_collection.EnsureIndex(mgo.Index{
			Key:        []string{"order_id"},
			Background: true,
			Unique:     true,
			DropDups:   true,
		})
	}
	if !is_index_key_present(indexes, []string{"order_state"}){
		orders_collection.EnsureIndex(mgo.Index{
			Key:        []string{"order_state"},
			Background: true,
			Unique:     false,
		})
	}

	orders_collection.EnsureIndex(mgo.Index{
		Key:        []string{"whom"},
		Background: true,
		Unique:     false,
	})
	orders_collection.EnsureIndex(mgo.Index{
		Key:        []string{"when"},
		Background: true,
		Unique:     false,
	})

	orders_collection.EnsureIndex(mgo.Index{
		Key:    []string{"source"},
		Background:true,
		Unique:false,
	})

	users_collection := session.DB(odbh.dbname).C("users")
	users_collection.EnsureIndex(mgo.Index{
		Key:        []string{"user_id"},
		Background: true,
		Unique:     true,
		DropDups:   true,
	})
	users_collection.EnsureIndex(mgo.Index{
		Key:        []string{"last_update"},
		Unique:     false,
		Background: true,
	})
	users_collection.EnsureIndex(mgo.Index{
		Key:        []string{"user_state"},
		Unique:     false,
		Background: true,
	})
	users_collection.EnsureIndex(mgo.Index{
		Key:        []string{"user_name"},
		Unique:     false,
		Background: true,
	})

	error_collection := session.DB(odbh.dbname).C("errors")

	error_collection.EnsureIndex(mgo.Index{
		Key: []string{"username"},
		Unique:false,
	})
	error_collection.EnsureIndex(mgo.Index{
		Key:[]string{"time"},
		Unique:false,
	})

	odbh.Users.collection = users_collection
	odbh.Orders.collection = orders_collection
	odbh.Errors.collection = error_collection
}

func NewDbHandler(conn, dbname string) *DbHandlerMixin {
	odbh := DbHandlerMixin{conn:conn, dbname:dbname}

	odbh.Users = &userHandler{parent:&odbh}
	odbh.Orders = &orderHandler{parent:&odbh}
	odbh.Errors = &errorHandler{parent:&odbh}

	odbh.Check = func() (string, bool) {
		if odbh.session != nil {
			return "OK", true
		}
		return "Db is not connected :(", false
	}
	log.Printf("start reconnecting")
	go func() {
		odbh.reConnect()
	}()
	return &odbh
}

func (oh *orderHandler) GetById(order_id int64, source string) (*OrderWrapper, error) {
	if oh.collection == nil {
		return nil, errors.New("БД не доступна")
	}
	result := OrderWrapper{}
	err := oh.collection.Find(bson.M{"order_id": order_id, "source":source}).One(&result)
	if err != nil && err != mgo.ErrNotFound {
		if err == io.EOF {
			oh.parent.reConnect()
			oh = oh.parent.Orders
		}
		return nil, err
	}
	return &result, nil
}

func (oh *orderHandler) SetState(order_id int64, source string, new_state int, order_data *OrderData) error {
	if oh.collection == nil {
		return errors.New("БД не доступна")
	}
	var to_set bson.M
	if order_data != nil {
		to_set = bson.M{"order_state": new_state, "when": time.Now(), "data": order_data}
	} else {
		to_set = bson.M{"order_state": new_state, "when": time.Now()}
	}
	change := bson.M{"$set": to_set}
	log.Println("change:", change["$set"])
	err := oh.collection.Update(bson.M{"order_id": order_id, "source":source}, change)
	if err != nil && err != mgo.ErrNotFound {
		return err
	}
	return nil
}

func (oh *orderHandler) SetFeedback(for_whom string, for_state int, feedback string, source string) (*int64, error) {
	if oh.collection == nil {
		return nil, errors.New("БД не доступна")
	}
	order := OrderWrapper{}
	err := oh.collection.Find(bson.M{"whom": for_whom, "order_state": for_state, "source":source}).Sort("-when").One(&order)
	if err != nil && err != mgo.ErrNotFound {
		return nil, err
	}
	if err == mgo.ErrNotFound {
		return nil, errors.New("Заказ не найден!")
	}
	err = oh.collection.Update(bson.M{"order_id": order.OrderId, "source":source}, bson.M{"$set": bson.M{"feedback": feedback}})
	order_id := order.OrderId
	return &order_id, err
}

func (oh *orderHandler) AddOrder(order_id int64, whom string, source string) error {
	if oh.collection == nil {
		return errors.New("БД не доступна")
	}
	wrapper := OrderWrapper{
		When:       time.Now(),
		Whom:       whom,
		OrderId:    order_id,
		OrderState: 1,
		Source: source,
	}
	err := oh.collection.Insert(&wrapper)
	return err
}

func (oh *orderHandler) AddOrderObject(order *OrderWrapper) error {
	if oh.collection == nil {
		return errors.New("БД не доступна")
	}
	order.When = time.Now()
	err := oh.collection.Insert(order)
	return err
}

func (oh *orderHandler) GetByOwner(whom, source string) (*OrderWrapper, error) {
	if oh.collection == nil {
		return nil, errors.New("БД не доступна")
	}
	result := OrderWrapper{}
	err := oh.collection.Find(bson.M{"whom": whom, "source":source}).Sort("-when").One(&result)
	if err == mgo.ErrNotFound {
		return nil, nil
	}else if err != nil {
		return nil, err
	}
	return &result, nil
}

func (oh *orderHandler) GetOrders(q bson.M) ([]OrderWrapper, error) {
	if oh.collection == nil {
		return nil, errors.New("БД не доступна")
	}
	var result []OrderWrapper
	err := oh.collection.Find(q).Sort("-when").One(&result)
	if err != nil && err != mgo.ErrNotFound {
		return nil, err
	}
	return result, nil
}

func (uh *userHandler) CheckUser(req bson.M) (*UserWrapper, error) {
	if uh.collection == nil {
		return nil, errors.New("БД не доступна")
	}
	tmp := UserWrapper{}
	err := uh.collection.Find(req).One(&tmp)
	if err == mgo.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, errors.New(fmt.Sprintf("user for %+v is not found", req))
	}
	return &tmp, nil
}

func (uh *userHandler) AddUser(user_id, phone *string) error {
	if uh.collection == nil {
		return errors.New("БД не доступна")
	}
	tmp, err := uh.CheckUser(bson.M{"user_id": user_id, "phone": phone})
	if tmp == nil {
		err = uh.collection.Insert(&UserWrapper{UserId: user_id, State: REGISTERED, Phone: phone, LastUpdate: time.Now()})
		return err
	}
	return nil
}

func (uh *userHandler) SetUserState(user_id *string, state string) error {
	if uh.collection == nil {
		return errors.New("БД не доступна")
	}
	tmp, _ := uh.CheckUser(bson.M{"user_id": user_id})
	if tmp == nil {
		err := uh.collection.Insert(&UserWrapper{UserId: user_id, State: state, LastUpdate: time.Now()})
		return err
	} else {
		err := uh.collection.Update(
			bson.M{"user_id": user_id},
			bson.M{"$set": bson.M{"user_state": state, "last_update": time.Now()}},
		)
		return err
	}
}

func (uh *userHandler) SetUserPassword(username, password *string) error {
	if uh.collection == nil {
		return errors.New("БД не доступна")
	}
	tmp, _ := uh.CheckUser(bson.M{"user_name": username})
	if tmp == nil {
		err := uh.collection.Insert(&UserWrapper{UserId: username, UserName: username, Password: password, State: REGISTERED, LastUpdate: time.Now()})
		return err
	} else if phash(password) != tmp.Password {
		log.Println("changing password! for user ", username)
		err := uh.collection.Update(
			bson.M{"user_name": username},
			bson.M{"$set": bson.M{"password": phash(password), "last_update": time.Now()}},
		)
		return err
	}
	return nil
}

func (uh *userHandler) GetUserState(user_id string) (*string, error) {
	if uh.collection == nil {
		return nil, errors.New("БД не доступна")
	}
	result := UserWrapper{}
	err := uh.collection.Find(bson.M{"user_id": user_id}).One(&result)
	return &(result.State), err
}

func (uh *userHandler) CheckUserPassword(username, password *string) (*bool, error) {
	if uh.collection == nil {
		return nil, errors.New("БД не доступна")
	}
	tmp := UserWrapper{}
	err := uh.collection.Find(bson.M{"user_name": username, "password": phash(password)}).One(&tmp)
	result := (err != nil)
	return &result, err
}

func (uh *userHandler) GetUserById(user_id string) (*UserWrapper, error) {
	if uh.collection == nil {
		return nil, errors.New("БД не доступна")
	}
	result := UserWrapper{}
	err := uh.collection.Find(bson.M{"user_id": user_id}).One(&result)
	return &result, err
}


func (eh *errorHandler) StoreError(username, error string) error {
	if eh.collection == nil {
		return errors.New("БД не доступна")
	}
	result := ErrorWrapper{Username:username, Error:error, Time:time.Now()}
	err := eh.collection.Insert(&result)
	return err
}