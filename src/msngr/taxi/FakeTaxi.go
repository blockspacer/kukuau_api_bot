package taxi

import (
	"log"
	"math/rand"
	"time"
	"encoding/json"
	c "msngr/configuration"
	"sync"
)

const FAKE = "fake"
//////////////////////////////////////////////////////////////////////////
///////THIS IS FAKE API FOR TEST//////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////

var fakeInstance *FakeTaxiAPI

func GetFakeAPI(params c.TaxiApiParams) TaxiInterface {
	if fakeInstance == nil {
		log.Printf("Create Fake API, params: %#v", params.Fake)
		fakeInstance = &FakeTaxiAPI{SleepTime:params.Fake.SleepTime, SendedStates:params.Fake.SendedStates}
	}
	return fakeInstance
}

type FakeTaxiAPI struct {
	sync.Mutex

	SleepTime    int
	SendedStates []int
	orders       []Order
}

func send_states(order_id int64, api *FakeTaxiAPI) {
	log.Printf("FA will send fake states for order %v", order_id, api.SendedStates)
	for _, i := range api.SendedStates {
		time.Sleep(time.Duration(api.SleepTime) * time.Second)
		api.set_order_state(order_id, i)
	}

	time.Sleep(time.Duration(api.SleepTime) * time.Second)
	api.delete_order(order_id)
}

func (inf *FakeTaxiAPI) set_order_state(order_id int64, new_state int) {
	for i, order := range inf.orders {
		if order.ID == order_id && order.State != ORDER_CANCELED {
			log.Printf("FA set state %v to order %v", new_state, order_id)
			inf.orders[i].State = new_state
			if rand.Intn(2)>0{
				log.Printf("FA set time before than now for order %v",order_id)
				ta := time.Now().Add(-10*time.Minute)
				if rand.Intn(2)>0{
					inf.orders[i].TimeArrival = &ta
				} else{
					inf.orders[i].TimeArrival = nil
					inf.orders[i].TimeDelivery = &ta
				}
			} else{
				ta := time.Now().Add(7*time.Minute)
				inf.orders[i].TimeDelivery = &ta
			}
		}
	}
}

func (inf *FakeTaxiAPI) delete_order(order_id int64){
	log.Printf("FA delete order [%v]", order_id)
	var oid int
	for i, order := range inf.orders{
		if order.ID == order_id{
			oid = i
		}
	}
	inf.orders = append(inf.orders[:oid], inf.orders[oid+1:]...)
}

func (inf *FakeTaxiAPI) NewOrder(order NewOrderInfo) Answer {
	saved_order := Order{
		ID:    rand.Int63(),
		State: 1,
		Cost:  150,
		IDCar: 5033615557,
	}
	result, _ := json.Marshal(order)
	log.Printf("FA New order:f\n%+v\n", string(result))

	inf.orders = append(inf.orders, saved_order)

	ans := Answer{
		IsSuccess: true,
		Message:   "test order was formed",
	}
	ans.Content.Id = saved_order.ID
	ans.Content.Cost = 150
	log.Println("FA now i have orders: ", len(inf.orders))

	go send_states(saved_order.ID, inf)

	return ans
}

func (inf *FakeTaxiAPI) Orders() []Order {
	inf.Lock()
	defer inf.Unlock()
	log.Printf("FA get orders... %+v", inf.orders)
	return inf.orders
}

func (inf *FakeTaxiAPI) CancelOrder(order_id int64) (bool, string, error) {
	log.Println("FA order was canceled", order_id)
	for i, order := range inf.orders {
		if order.ID == order_id {
			inf.orders[i].State = ORDER_CANCELED
			return true, "test order was cancelled", nil
		}
	}
	return true, "Test order not found :( ", nil
}

func (p *FakeTaxiAPI) CalcOrderCost(order NewOrderInfo) (int, string) {
	log.Println("FA calulate cost for order: ", order)
	return 100500, "Good cost!"
}

func (p *FakeTaxiAPI) Feedback(f Feedback) (bool, string) {
	return true, "Test feedback was received! Thanks!"
}

func (p *FakeTaxiAPI) IsConnected() bool {
	return true
}

func (p *FakeTaxiAPI) GetCarsInfo() []CarInfo {
	return []CarInfo{
		CarInfo{
			ID:5033615557,
			Number:"X777XX",
			Color:"ультрамариновый",
			Model:"Боливар",
		},
	}
}

func (p *FakeTaxiAPI) WriteDispatcher(message string) (bool, string) {
	log.Printf("I have new message: %s", message)
	return true, "Мессадж доставлен. Окстись."
}

func (p *FakeTaxiAPI) CallbackRequest(phone string) (bool, string) {
	log.Printf("I must call to: %s", phone)
	return true, "Вам перезвонят."
}

func (p *FakeTaxiAPI) WhereIt(order_id int64) (bool, string) {
	log.Printf("Whre it for %v", order_id)
	return true, "Водитель вас тоже не видет. Покрутитеся вокруг, авось чего узреите."
}

func (p *FakeTaxiAPI) Markups() []Markup {
	return []Markup{Markup{Name:"Животное", Value:100500, ID:1234567890}}
}

func (p *FakeTaxiAPI) Connect(){

}