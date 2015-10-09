package infinity

import (
	"encoding/json"
	"errors"

	"io/ioutil"
	"log"
	"net/http"

	"time"
	t "msngr/taxi"
)

func warnp(err error) {
	if err != nil {
		log.Println(err)
		panic(err)
	}
}

// infinity - Структура для работы с API infinity.
type infinity struct {
	Host          string
	ConnString    string // Строка подключения к infinity API// default: http://109.202.25.248:8080/WebAPITaxi/
	LoginTime     time.Time
	Cookie        *http.Cookie
	LoginResponse struct {
					  Success   bool  `json:"success"`
					  IDClient  int64 `json:"idClient"`
					  Params    struct {
									ProtocolVersion            int    `json:"ProtocolVersion"`
									RefreshOrdersSeconds       int    `json:"RefreshOrdersSeconds"`
									LoginRegEx                 string `json:"LoginRegEx"`
									MyPhoneRegEx               string `json:"MyPhoneRegEx"`
									OurPhoneDisplay            string `json:"OurPhoneDisplay"`
									OurPhoneNumber             string `json:"OurPhoneNumber"`
									DefaultInfinityServiceID   int64  `json:"DefaultInfinityServiceID"`
									DefaultInfinityServiceName string `json:"DefaultInfinityServiceName"`
									DefaultRegionID            int64  `json:"DefaultRegionID"`
									DefaultRegionName          string `json:"DefaultRegionName"`
									DefaultDistrictID          string `json:"DefaultDistrictID"` // Can be null, so used as string here.
									DefaultDistrictName        string `json:"DefaultDistrictName"`
									DefaultCityID              int64  `json:"DefaultCityID"`
									DefaultCityName            string `json:"DefaultCityName"`
									DefaultPlaceID             string `json:"DefaultPlaceID"`    // Can be null, so used as string here.
									DefaultPlaceName           string `json:"DefaultPlaceName"`
								} `json:"params"`
					  SessionID string `json:"sessionid"`
				  }
	Message       struct {
					  Success bool   `json:"isSuccess"`
					  Content string `json:"content"`
				  }
	Services      []InfinityServices `json:"InfinityServices"`

	Config        t.TaxiAPIConfig
}

// Global API variable
var instance *infinity


func _initInfinity(config t.TaxiAPIConfig) *infinity {
	result := &infinity{}
	result.ConnString = config.GetConnectionString()
	result.Host = config.GetConnectionString()
	result.Config = config

	logon := result.Login(config.GetLogin(), config.GetPassword())

	if !logon {
		go result.reconnect()
	}

	return result
}

func GetInfinityAPI(tc t.TaxiAPIConfig) t.TaxiInterface {
	if instance == nil {
		instance = _initInfinity(tc)
	}
	return instance
}

func GetInfinityAddressSupplier(tc t.TaxiAPIConfig) t.AddressSupplier {
	if instance == nil {
		instance = _initInfinity(tc)
	}
	return instance
}

// Login - Авторизация в сервисе infinity. Входные параметры: login:string; password:string.
// Возвращает true, если авторизация прошла успешно, false иначе.
// Устанавливает время авторизации в infinity.LoginTime при успешной авторизации.
func (p *infinity) Login(login, password string) bool {
	p.LoginResponse.Success = false

	client := &http.Client{}
	req, err := http.NewRequest("GET", p.ConnString + "Login", nil)
	warnp(err)
	req.Header.Add("ContentType", "text/html;charset=UTF-8")

	values := req.URL.Query()
	values.Add("l", login)
	values.Add("p", password)
	values.Add("app", "CxTaxiClient")
	req.URL.RawQuery = values.Encode()
	res, err := client.Do(req)

	//если нет соединения с infinity то выходим
	if err != nil {
		return false
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	warnp(err)
	err = json.Unmarshal(body, &p.LoginResponse)
	warnp(err)
	log.Printf("[login] self: %+q\n", p)
	if p.LoginResponse.Success {
		log.Println("[login] JSESSIONID: ", p.LoginResponse.SessionID)
		p.Cookie = &http.Cookie{
			Name:   "JSESSIONID",
			Value:  p.LoginResponse.SessionID,
			Path:   "/",
			Domain: "109.202.25.248",
		}
		p.LoginTime = time.Now()

		return true
	}
	return false
}

func (p *infinity) IsConnected() bool {
	return p.LoginResponse.Success
}

func (p *infinity) reconnect() {
	if p.Config.GetLogin() == "" && p.Config.GetPassword() == "" {
		panic(errors.New("reconnect before connect! I don't know login and password :( "))
	}
	sleep_time := time.Duration(1000)
	for {
		result := p.Login(p.Config.GetLogin(), p.Config.GetPassword())
		if result {
			break
		} else {
			log.Printf("IR: reconnect is fail trying next after %+v", sleep_time)
			time.Sleep(sleep_time * time.Millisecond)
			sleep_time = time.Duration(float32(sleep_time) * 1.4)
		}
	}
}

// Ping возвращает true если запрос выполнен успешно и время сервера infinity в формате yyyy-MM-dd HH:mm:ss.
// Если запрос выполнен неуспешно возвращает false и пустую строку.
// Условие: пользователь должен быть авторизован.
func (p *infinity) Ping() (bool, string) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", p.ConnString + "RemoteCall", nil)
	warnp(err)
	req.Header.Add("ContentType", "text/html;charset=UTF-8")
	values := req.URL.Query()
	values.Add("method", "Taxi.WebAPI.GetDateTime")
	req.URL.RawQuery = values.Encode()

	//log.Println(req.URL)

	req.AddCookie(p.Cookie)
	//log.Println("Cookies in request? ", req.Cookies())
	res, err := client.Do(req)
	if res.Status == "403 Forbidden" {
		err = errors.New("Ошибка авторизации infinity! (Возможно не установлены cookies)")
		p.reconnect()
	}
	warnp(err)
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	warnp(err)

	//log.Println(string(body))
	err = json.Unmarshal(body, &p.Message)
	warnp(err)
	return p.Message.Success, p.Message.Content
}

type InfinityService struct {
	ID                 int64  `json:"id"`
	Name               string `json:"name"`
	Description        string `json:"description"`
	AvailableToClients bool   `json:"AvailableToClients"`
}
type InfinityServices struct {
	Rows []InfinityService `json:"rows"`
}
type InfinityCarsInfo struct {
	Rows []t.CarInfo `json:"rows"`
}

func (p *infinity) _request(conn_suffix string, url_values map[string]string) []byte {
	req, err := http.NewRequest("GET", p.ConnString + conn_suffix, nil)
	warnp(err)
	req.Header.Add("ContentType", "text/html;charset=UTF-8")
	values := req.URL.Query()
	for k, v := range url_values {
		values.Add(k, v)
	}

	req.URL.RawQuery = values.Encode()
	req.AddCookie(p.Cookie)

	client := &http.Client{}
	res, err := client.Do(req)
	if res == nil || err != nil {
		log.Println("INF response is: ", res, "; error is:", err, ". I will reconnect and will retrieve data again after 3s.")
		time.Sleep(3 * time.Second)
		p.reconnect()
		return p._request(conn_suffix, url_values)
	}
	if res.Status == "403 Forbidden" {
		err = errors.New("Ошибка авторизации infinity! (Возможно не установлены cookies)")
		p.reconnect()
	}
	warnp(err)
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	warnp(err)
	return body
}

// GetServices возвращает информацию об услугах доступных для заказа (filterField is set to true!)
func (p *infinity) GetServices() []InfinityService {
	var tmp []InfinityServices

	body := p._request("GetViewData", map[string]string{"params": "[{\"viewName\":\"Taxi.Services\",\"filterField\":{\"n\":\"AvailableToClients\",\"v\":true}}]"})
	err := json.Unmarshal(body, &tmp)
	warnp(err)
	return tmp[0].Rows
}

// GetCarsInfo возвращает информацию о машинах
func (p *infinity) GetCarsInfo() []t.CarInfo {
	var tmp []InfinityCarsInfo
	body := p._request("GetViewData", map[string]string{"params": "[{\"viewName\":\"Taxi.Cars.InfoEx\"}]"})
	err := json.Unmarshal(body, &tmp)
	warnp(err)
	return tmp[0].Rows
}

func (p *infinity) NewOrder(order t.NewOrder) t.Answer {
	log.Printf("INF NO: %+v", order)
	order.IdService = p.Config.GetIdService()
	param, err := json.Marshal(order)
	warnp(err)
	log.Printf("INF NO jsonified: %+v", string(param))
	body := p._request("RemoteCall", map[string]string{"params": string(param), "method": "Taxi.WebAPI.NewOrder"})
	var ans t.Answer
	err = json.Unmarshal(body, &ans)
	warnp(err)
	return ans
}

func (p *infinity) CalcOrderCost(order t.NewOrder) (int, string) {
	param, err := json.Marshal(order)
	warnp(err)
	body := p._request("RemoteCall", map[string]string{"params": string(param), "method": "Taxi.WebAPI.CalcOrderCost"})
	var tmp t.Answer
	err = json.Unmarshal(body, &tmp)
	warnp(err)
	return tmp.Content.Cost, tmp.Content.Details
}

//{"phone":"89261234567","deliveryTime":"2015-07-15+07:00:00","deliveryMinutes":60,"idService":7006261161,"notes":"Хочется+комфортную+машину","markups":[7002780031,7004760103],"attributes":[1000113000,1000113002],"delivery":{"idRegion":7006803034,"idStreet":0,"house":"1","fraction":"1","entrance":"2","apartment":"30"},"destinations":{["lat":55.807898,"lon":37.785449,"idRegion":7006803034,"idPlace":7006803054,"idStreet":7006803054,"house":"12","entrance":"2","apartment":"30"}]}

type PrivateParams struct {
	Name  string `json:"name"`
	Login string `json:"login"`
}

//Taxi.WebAPI.Client.GetPrivateParams (Получение параметров клиента)
//Контент:
//Параметры личного кабинета клиента в виде JSON объекта: { "name" : <Имя клиента>, "login" : <Логин клиента> }
func (p *infinity) GetPrivateParams() (bool, string, string) {

	body := p._request("RemoteCall", map[string]string{"method": "Taxi.WebAPI.Client.GetPrivateParams"})
	var temp t.Answer
	err := json.Unmarshal(body, &temp)
	warnp(err)
	return temp.IsSuccess, temp.Content.Name, temp.Content.Login
}

/////////////////////////////////////////////////////////////////////////////////////////////////////
/////////////////////////////////////////////////////////////////////////////////////////////////////
/////////////////////////////////////////////////////////////////////////////////////////////////////

//Taxi.WebAPI.Client.ChangePassword (Изменение пароля) Изменяет пароль клиента.
//Параметры:
//Новый пароль (строка)
func (p *infinity) ChangePassword(password string) (bool, string) {
	tmp, err := json.Marshal(password)
	warnp(err)
	body := p._request("RemoteCall", map[string]string{"params": string(tmp), "method": "Taxi.WebAPI.Client.ChangePassword"})
	var temp t.Answer
	err = json.Unmarshal(body, &temp)
	warnp(err)
	return temp.IsSuccess, temp.Message
}

//Taxi.WebAPI.Client.ChangeName (Изменение имени клиента) Изменяет имя клиента в системе.
//Параметры:
//Новое имя клиента (строка)
func (p *infinity) ChangeName(name string) (bool, string) {

	tmp, err := json.Marshal(name)
	warnp(err)

	body := p._request("RemoteCall", map[string]string{"params": string(tmp), "method": "Taxi.WebAPI.Client.ChangeName"})

	var temp t.Answer
	err = json.Unmarshal(body, &temp)
	warnp(err)
	return temp.IsSuccess, temp.Message
}

//Taxi.WebAPI.Client.SendMessage (Отправка сообщения оператору) Отправляет операторам системы уведомление с сообщением данного клиента
//Параметры:
//Текст сообщения (строка)
func (p *infinity) SendMessage(message string) (bool, string /*, string*/) {
	tmp, err := json.Marshal(message)
	warnp(err)

	body := p._request("RemoteCall", map[string]string{"params": string(tmp), "method": "Taxi.WebAPI.Client.SendMessage"})

	var temp t.Answer
	err = json.Unmarshal(body, &temp)
	warnp(err)
	return temp.IsSuccess, temp.Message
}

func (p *infinity) CallbackRequest(phone string) (bool, string) {
	tmp, err := json.Marshal(phone)
	warnp(err)
	body := p._request("RemoteCall", map[string]string{"params": string(tmp), "method": "Taxi.WebAPI.Client.CallbackRequest"})
	var temp t.Answer
	err = json.Unmarshal(body, &temp)
	warnp(err)
	return temp.IsSuccess, temp.Message
}

//Taxi.WebAPI.Client.ClearHistory (Очистка истории заказов клиента)
//Отмечает закрытые заказы клиента как не видимые для личного кабинета (т.е. сама информация о заказе не удаляется)
func (p *infinity) ClearHistory() (bool, string) {
	body := p._request("RemoteCall", map[string]string{"method": "Taxi.WebAPI.Client.ClearHistory"})

	var temp t.Answer
	err := json.Unmarshal(body, &temp)
	warnp(err)
	return temp.IsSuccess, temp.Message
}

//Taxi.WebAPI.Client.CancelOrder (Отказ от заказа) Устанавливает для указанного заказа состояние «Отменен»
//Параметры:
//Идентификатор заказа (Int64)
func (p *infinity) CancelOrder(order int64) (bool, string) {
	tmp, err := json.Marshal(order)
	warnp(err)

	body := p._request("RemoteCall", map[string]string{"params": string(tmp), "method": "Taxi.WebAPI.Client.CancelOrder"})

	var temp t.Answer
	err = json.Unmarshal(body, &temp)
	warnp(err)
	return temp.IsSuccess, temp.Message
}

//Taxi.WebAPI.Client.Feedback (Отправка отзыва о заказе)
//Указывает оценку и отзыв для указанного заказа, отправляя операторам системы уведомления об отзыве.
//Параметры:
//JSON объект: {
//"idOrder" : <Идентификатор заказа (Int64)>,
//"rating" : <Оценка (число)>,
//"notes" : <Текст отзыва>
//}


func (p *infinity) Feedback(inf t.Feedback) (bool, string) {
	tmp, err := json.Marshal(inf)
	warnp(err)

	body := p._request("RemoteCall", map[string]string{"params": string(tmp), "method": "Taxi.WebAPI.Client.Feedback"})

	var temp t.Answer
	err = json.Unmarshal(body, &temp)
	warnp(err)
	return temp.IsSuccess, temp.Message
}

//Taxi.WebAPI.Client.WhereIT (Отправка запроса «Клиент не видит машину»)
//Отправляет операторам системы уведомление «Клиент не видит машину»
//Параметры:
//Идентификатор заказа (Int64)
func (p *infinity) WhereIT(ID int64) (bool, string) {
	tmp, err := json.Marshal(ID)
	warnp(err)

	body := p._request("RemoteCall", map[string]string{"params": string(tmp), "method": "Taxi.WebAPI.Client.WhereIT"})

	var temp t.Answer
	err = json.Unmarshal(body, &temp)
	warnp(err)
	return temp.IsSuccess, temp.Message
}

//Taxi.WebAPI.Client.Phones.Edit (Изменение/ Добавление телефона клиента)
//Изменяет телефон клиента, если в параметрах указан идентификатор существующего телефона данного
//клиента.
//Добавляет новый телефон клиента, если в параметрах отсутствует идентификатор существующего телефона.
//Параметры:
//JSON объект: {
//"id" : <Идентификатор телефона (Int64), необходим при редактировании>,
//"contact" : <Номер телефона (строка)>
//}

type phonesEdit struct {
	Id      int64  `json:"id"`
	Contact string `json:"contact"`
}

func (p *infinity) PhonesEdit(phone phonesEdit) (bool, string) {
	tmp, err := json.Marshal(phone)
	warnp(err)
	body := p._request("RemoteCall", map[string]string{"params": string(tmp), "method": "Taxi.WebAPI.Client.Phones.Edit"})
	var temp t.Answer
	err = json.Unmarshal(body, &temp)
	warnp(err)
	return temp.IsSuccess, temp.Message
}

//Taxi.WebAPI.Client.Phones.Remove (Удаление телефона клиента) Удаляет указанный телефон клиента.
//Параметры:
//Идентификатор телефона клиента (Int64)
func (p *infinity) PhonesRemove(phone int64) (bool, string) {
	tmp, err := json.Marshal(phone)
	warnp(err)
	body := p._request("RemoteCall", map[string]string{"params": string(tmp), "method": "Taxi.WebAPI.Client.Phones.Remove"})
	var temp t.Answer
	err = json.Unmarshal(body, &temp)
	warnp(err)
	return temp.IsSuccess, temp.Message
}

//Taxi.WebAPI.Client.Addresses.Edit (Изменение/ Добавление адреса клиента)
//Изменяет «любимый» адрес клиента, если в параметрах указан идентификатор существующего элемента
//справочника, в противном случае будет добавлен новый адрес клиента.
type favorite struct {
	Id         int64  `json:"id"`         // <Идентификатор любимогo адреса (Int64)>,
	Name       string `json:"name"`       // <Наименование элемента (строка)>,
	ImageIndex int    `json:"imageIndex"` // <Индекс иконки, адреса (число)>,
	IdAddres   string `json:"idAddress"`  // <Идентификатор существующего описания адреса (адрес дома или объекта)>,
	IdRedion   int64  `json:"idRegion"`   // <Идентификатор региона (Int64)>,
	IdDistrict int64  `json:"idDistrict"` // <Идентификатор района (Int64)>,
	IdCity     int64  `json:"idCity"`     // <Идентификатор города (Int64)>,
	IdPlace    int64  `json:"idPlace"`    // <Идентификатор поселения (Int64)>,
	IdStreet   int64  `json:"idStreet"`   // <Идентификатор улицы (Int64)>,
	House      string `json:"house"`      // <No дома (строка)>,
	Building   string `json:"building"`   // <Строение (строка)>,
	Fracion    string `json:"fraction"`   // <Корпус (строка)>,
	Entrance   string `json:"entrance"`   // <Подъезд (строка)>,
	Apartament string `json:"apartment"`  // <No квартиры (строка)>
}

//Параметры idRegion, idDistrict, idCity, idStreet, house, building, fraction используются для создания нового
//описания адреса и не анализируются при указании параметра idAddress.
func (p *infinity) AddressesEdit(f favorite) (bool, string) {

	tmp, err := json.Marshal(f)
	warnp(err)

	body := p._request("RemoteCall", map[string]string{"params": string(tmp), "method": "Taxi.WebAPI.Client.Addresses.Edit"})
	var temp t.Answer
	err = json.Unmarshal(body, &temp)
	warnp(err)

	return temp.IsSuccess, temp.Message
}

func (p *infinity) AddressesRemove(id int64) (bool, string) {
	tmp, err := json.Marshal(id)
	warnp(err)

	body := p._request("RemoteCall", map[string]string{"params": string(tmp), "method": "Taxi.WebAPI.Client.Addresses.Remove"})
	var temp t.Answer
	err = json.Unmarshal(body, &temp)
	warnp(err)
	return temp.IsSuccess, temp.Message
}

/////////////////////////////
type Orders struct {
	Rows []t.Order `json:"rows"`
}

//Taxi.t.Orders (Заказы: активные и предварительные)
func (p *infinity) Orders() []t.Order {
	body := p._request("GetViewData", map[string]string{"params": "[{\"viewName\": \"Taxi.Orders\"}]"})
	temp := []Orders{}
	//	log.Println(">>>", string(body))
	err := json.Unmarshal(body, &temp)
	warnp(err)
	//	log.Printf(">>> umshld len:%v,\n %+v,", len(temp[0].Rows), temp[0].Rows)
	return temp[0].Rows
}

//Taxi.t.Orders.Closed.ByDates (История заказов: По датам)
func (p *infinity) OrdersClosedByDates() []t.Order {
	body := p._request("GetViewData", map[string]string{"params": "[{\"viewName\": \"Taxi.Orders.Closed.ByDates\"}]"})
	temp := []Orders{}
	err := json.Unmarshal(body, &temp)
	warnp(err)
	return temp[0].Rows
}

//Taxi.Orders.Closed.LastN (История заказов: Последние)
func (p *infinity) OrdersClosedlastN() []t.Order {
	body := p._request("GetViewData", map[string]string{"params": "[{\"viewName\": \"Taxi.Orders.Closed.LastN\"}]"})

	var temp []t.Order
	err := json.Unmarshal(body, &temp)
	warnp(err)
	return temp
}

//Taxi.Destinations.ByActivet.Order (Пункты назначения: Активные заказы)
//Taxi.Destinations.ByClosedt.Order (Пункты назначения: Закрытые заказы (история))

//Taxi.Markups (Список доступных наценок)
func (p *infinity) Markups() []t.Order {
	body := p._request("GetViewData", map[string]string{"params": "[{\"viewName\": \"Taxi.Markups\"}]"})

	var temp []t.Order
	err := json.Unmarshal(body, &temp)
	warnp(err)
	return temp
}

//Taxi.Services (Список услуг)
//Taxi.ClientPhones (Телефоны клиента)
//Taxi.Cars.Info (Дополнительная информация о машине)
//Taxi.CarAttributes (Список атрибутов машины)

/////////////////////////////////////////////////////////////////////////////////////////////////////
/////////////////////////////////////////////////////////////////////////////////////////////////////
/////////////////////////////////////////////////////////////////////////////////////////////////////
//Taxi.t.FastAddresses.Search (Поиск быстрых адресов) Доступность: Личный кабинет + Заказ с сайта
//Поля:  ID  Name  IDType  IDAddress  Apartment  Entrance  StrAddress  AddrDescription  Type
//Наименование
//Тип адреса/быстрого адреса
//ID адреса
//No квартиры (строка)
//Подъезд
//Фактический адрес
//Описание адреса
//Тип адреса/быстрого адреса в виде строки


func (p *infinity) AddressesSearch(text string) t.FastAddress {
	body := p._request("GetViewData", map[string]string{"params": "[{\"viewName\": \"Taxi.Addresses.Search\", \"params\": [{\"n\": \"SearchText\", \"v\": \"" + text + "\"}]}]"})
	var temp []t.FastAddress
	err := json.Unmarshal(body, &temp)
	warnp(err)
	return temp[0]
}

//Taxi.ClientAddresses (Адреса клиента)
func (p *infinity) ClientAddresses() t.FastAddress {
	body := p._request("GetViewData", map[string]string{"params": "[{\"viewName\": \"Taxi.ClientAddresses\"}]"})
	var temp []t.FastAddress
	err := json.Unmarshal(body, &temp)
	warnp(err)
	return temp[0]
}

var StatusesMap = map[int]string{
	1:  "Не распределен",
	2:  "Назначен",
	3:  "Выехал",
	4:  "Ожидание клиента",
	5:  "Выполнение",
	6:  "Простой",
	7:  "Оплачен",
	8:  "Не оплачен",
	9:  "Отменен",
	11: "Запланирована машина",
	12: "Зафиксирован",
	13: "Не создан",
	14: "Горящий заказ",
	15: "Не подтвержден",
}
