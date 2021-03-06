package quests

import (
	"github.com/go-martini/martini"
	"github.com/martini-contrib/auth"
	"github.com/martini-contrib/render"
	"github.com/tealeg/xlsx"

	c "msngr/configuration"

	ntf "msngr/notify"

	"log"
	"gopkg.in/mgo.v2/bson"
	"net/http"
	"fmt"
	"strings"
	"io/ioutil"
	"strconv"
	"regexp"
	"html/template"
	"encoding/json"
	"time"
	"msngr/utils"
	w "msngr/web"
	"errors"
	"github.com/martini-contrib/binding"
)

var users = map[string]string{
	"alesha":"sederfes100500",
	"leha":"qwerty100500",
	"dima":"123",
}

const (
	ALL = "all"
	ALL_TEAM_MEMBERS = "all_team_members"
)

func ValidateKeys(kv [][]string) (map[string]string, error) {
	teams := map[string][]string{}
	result := map[string]string{}
	for _, v := range kv {
		start := v[0]
		next := v[2]
		team_name, tns_err := GetTeamNameFromKey(start)
		tn_next, tnn_err := GetTeamNameFromKey(next)
		if tnn_err != nil && tns_err != nil {
			return result, errors.New(fmt.Sprintf("Не могу определить комманду из ключа %v или %v", start, next))
		}
		if team_name != tn_next && team_name != "" && tn_next != "" {
			return result, errors.New(fmt.Sprintf("Для шага %v -> %v разные комманды.", start, next))
		}

		if tkeys, ok := teams[team_name]; ok {
			teams[team_name] = append(tkeys, start)
		} else {
			teams[team_name] = []string{start}
		}
	}

	for k, v := range teams {
		result[k] = strings.Join(v, " > ")
	}
	return result, nil
}

func GetKeysErrorInfo(err_text string, qs *QuestStorage) map[string]interface{} {
	var e error
	result := map[string]interface{}{}

	keys, e := qs.GetAllSteps()

	if e != nil || err_text != "" {
		result["is_error"] = true
		if e != nil {
			result["error_text"] = e.Error()
		} else {
			result["error_text"] = err_text
		}
	}
	result["keys"] = keys
	return result
}

func SortSteps(steps []Step) []Step {
	step_map_next := map[string]Step{}
	step_map_start := map[string]Step{}
	sorted := []Step{}
	var first_step Step
	for _, step := range steps {
		step_map_next[step.NextKey] = step
		step_map_start[step.StartKey] = step
	}
	for _, step := range steps {
		if _, ok := step_map_next[step.StartKey]; !ok {
			first_step = step
		}
	}
	//log.Printf("QS start key: %+v, \nstep_map_next: %+v\nstep_map_start %+v", first_step, step_map_next, step_map_start)
	sorted = append(sorted, first_step)
	for _, _ = range steps {
		if next_step, ok := step_map_start[first_step.NextKey]; ok {
			sorted = append(sorted, next_step)
			first_step = next_step
		} else {
			break
		}
	}
	return sorted
}

func GetKeysTeamsInfo(teams_info map[string]string, qs *QuestStorage) map[string]interface{} {
	result := map[string]interface{}{}
	keys, _ := qs.GetAllSteps()
	result["keys"] = keys
	result["is_team_info"] = true
	result["team_info"] = teams_info
	return result
}

func SendMessagesToPeoples(people []TeamMember, ntf *ntf.Notifier, text string) {
	go func() {
		for _, user := range people {
			ntf.NotifyText(user.UserId, text)
		}
	}()
}

func Run(config c.QuestConfig, qs *QuestStorage, ntf *ntf.Notifier, additionalNotifier *ntf.Notifier) {
	m := martini.New()
	m.Use(w.NonJsonLogger())
	m.Use(martini.Recovery())
	m.Use(render.Renderer(render.Options{
		Directory:"templates/quests",
		Layout: "layout",
		Extensions: []string{".tmpl", ".html"},
		Charset: "UTF-8",
		IndentJSON: true,
		IndentXML: true,
		Funcs:[]template.FuncMap{
			template.FuncMap{
				"eq_s":func(a, b string) bool {
					return a == b
				},
				"stamp_date":func(t time.Time) string {
					return t.Format(time.Stamp)
				},
			},
		},
	}))

	m.Use(auth.BasicFunc(func(username, password string) bool {
		pwd, ok := users[username]
		return ok && pwd == password
	}))

	m.Use(martini.Static("static"))

	r := martini.NewRouter()

	r.Get("/", func(user auth.User, render render.Render) {
		render.HTML(200, "readme", map[string]interface{}{})
	})

	r.Get("/new_keys", func(render render.Render) {
		render.HTML(200, "new_keys", GetKeysErrorInfo("", qs))
	})

	r.Post("/add_key", func(user auth.User, render render.Render, request *http.Request) {
		start_key := strings.TrimSpace(request.FormValue("start-key"))
		next_key := strings.TrimSpace(request.FormValue("next-key"))
		description := request.FormValue("description")

		log.Printf("QUESTS WEB add key %s -> %s -> %s", start_key, description, next_key)
		if start_key != "" && description != "" {
			key, err := qs.AddStep(start_key, description, next_key)
			if key != nil &&err != nil {
				render.HTML(200, "new_keys", GetKeysErrorInfo("Такой ключ уже существует. Используйте изменение ключа если хотите его изменить.", qs))
				return
			}
		} else {

			render.HTML(200, "new_keys", GetKeysErrorInfo("Невалидные значения ключа или ответа", qs))
			return
		}
		render.Redirect("/new_keys")
	})

	r.Post("/delete_key/:key", func(params martini.Params, render render.Render) {
		key := params["key"]
		err := qs.DeleteStep(key)
		log.Printf("QUESTS WEB will delete %v (%v)", key, err)
		render.Redirect("/new_keys")
	})

	r.Post("/update_key/:key", func(params martini.Params, render render.Render, request *http.Request) {
		key_id := params["key"]

		start_key := strings.TrimSpace(request.FormValue("start-key"))
		next_key := strings.TrimSpace(request.FormValue("next-key"))
		description := request.FormValue("description")

		err := qs.UpdateStep(key_id, start_key, description, next_key)
		log.Printf("QUESTS WEB was update key %s %s %s %s\n err? %v", key_id, start_key, description, next_key, err)
		render.Redirect("/new_keys")
	})

	r.Get("/delete_key_all", func(render render.Render) {
		qs.Steps.RemoveAll(bson.M{})
		render.Redirect("/new_keys")
	})

	xlsFileReg := regexp.MustCompile(".+\\.xlsx?")

	r.Post("/load/up", func(render render.Render, request *http.Request) {
		file, header, err := request.FormFile("file")

		log.Printf("QS: Form file information: file: %+v \nheader:%v, %v\nerr:%v", file, header.Filename, header.Header, err)

		if err != nil {
			render.HTML(200, "new_keys", GetKeysErrorInfo(fmt.Sprintf("Ошибка загрузки файлика: %v", err), qs))
			return
		}
		defer file.Close()

		data, err := ioutil.ReadAll(file)
		if err != nil {
			render.HTML(200, "new_keys", GetKeysErrorInfo(fmt.Sprintf("Ошибка загрузки файлика: %v", err), qs))
			return
		}

		if xlsFileReg.MatchString(header.Filename) {
			xlFile, err := xlsx.OpenBinary(data)

			if err != nil || xlFile == nil {
				render.HTML(200, "new_keys", GetKeysErrorInfo(fmt.Sprintf("Ошибка обработки файлика: %v", err), qs))
				return
			}
			skip_rows, errsr := strconv.Atoi(request.FormValue("skip-rows"))
			skip_cols, errsc := strconv.Atoi(request.FormValue("skip-cols"))
			if errsr != nil || errsc != nil {
				render.HTML(200, "new_keys", GetKeysErrorInfo("Не могу распознать количества столбцов и строк пропускаемых :(", qs))
				return
			}
			log.Printf("QS: Will process file: %+v, err: %v \n with skipped rows: %v, cols: %v", xlFile, err, skip_rows, skip_cols)
			parse_res, errp := w.ParseExportXlsx(xlFile, skip_rows, skip_cols)
			if errp != nil {
				render.HTML(200, "new_keys", GetKeysErrorInfo("Ошибка в парсинге файла:(", qs))
				return
			}
			res, val_err := ValidateKeys(parse_res)
			if val_err != nil {
				render.HTML(200, "new_keys", GetKeysErrorInfo(val_err.Error(), qs))
				return
			}
			for _, prel := range parse_res {
				qs.AddStep(prel[0], prel[1], prel[2])
			}
			render.HTML(200, "new_keys", GetKeysTeamsInfo(res, qs))
		} else {
			render.HTML(200, "new_keys", GetKeysErrorInfo("Файл имеет не то расширение :(", qs))
		}

		render.Redirect("/new_keys")
	})

	r.Get("/chat", func(render render.Render, params martini.Params, req *http.Request) {
		var with string
		result_data := map[string]interface{}{}
		query := req.URL.Query()
		for key, value := range query {
			if key == "with" && len(value) > 0 {
				with = value[0]
				log.Printf("QSERV: with found is: %v", with)
				break
			}
		}
		type Collocutor struct {
			IsTeam   bool
			IsMan    bool
			IsAll    bool
			IsWinner bool
			WinTime  string
			Info     interface{}
			Name     string
		}
		collocutor := Collocutor{}

		var messages []Message

		if with != ALL && with != ALL_TEAM_MEMBERS {
			if team, _ := qs.GetTeamByName(with); team != nil {
				type TeamInfo struct {
					FoundedKeys []string
					Members     []TeamMember
					AllKeys     []Step
				}

				collocutor.Name = team.Name
				collocutor.IsTeam = true
				collocutor.IsWinner = team.Winner
				if collocutor.IsWinner {
					tm := time.Unix(team.WinTime, 0)
					collocutor.WinTime = tm.Format("Mon 15:04:05")
				}
				members, _ := qs.GetMembersOfTeam(team.Name)
				keys, _ := qs.GetSteps(bson.M{"for_team":team.Name})
				keys = SortSteps(keys)
				collocutor.Info = TeamInfo{FoundedKeys:team.FoundKeys, Members:members, AllKeys:keys}

				messages, _ = qs.GetMessages(bson.M{
					"$or":[]bson.M{
						bson.M{"from":team.Name},
						bson.M{"to":team.Name},
					},
				})
			} else {
				if peoples, _ := qs.GetPeoples(bson.M{"user_id":with}); len(peoples) > 0 {
					man := peoples[0]
					collocutor.IsMan = true
					collocutor.Name = man.Name
					collocutor.Info = man

					messages, _ = qs.GetMessages(bson.M{
						"$or":[]bson.M{
							bson.M{"from":man.UserId},
							bson.M{"to":man.UserId},
						},
					})
					for i, _ := range messages {
						if messages[i].From != ME {
							messages[i].From = man.Name
						}
					}
				} else {
					with = "all"
				}
			}
		}

		if strings.HasPrefix(with, "all") {
			collocutor.IsAll = true
			collocutor.Name = with
			messages, _ = qs.GetMessages(bson.M{"to":with})
		}

		//log.Printf("QS i return this messages: %+v", messages)
		result_data["with"] = with
		result_data["collocutor"] = collocutor
		if len(messages) > 100 {
			messages = messages[0:100]
		}
		result_data["messages"] = messages

		qs.SetMessagesRead(with)

		all_teams, _ := qs.GetAllTeams()
		if contacts, err := qs.GetContacts(all_teams); err == nil {
			result_data["contacts"] = contacts
		}

		render.HTML(200, "chat", result_data)
	})

	r.Post("/send", func(render render.Render, req *http.Request) {
		type MessageFromF struct {
			From string `json:"from"`
			To   string `json:"to"`
			Body string `json:"body"`
		}
		data, err := ioutil.ReadAll(req.Body)
		if err != nil {
			log.Printf("QS QE E: errror at reading req body %v", err)
			render.JSON(500, map[string]interface{}{"error":err})
			return
		}
		message := MessageFromF{}
		err = json.Unmarshal(data, &message)
		if err != nil {
			log.Printf("QS QE E: at unmarshal json messages %v\ndata:%s", err, data)
			render.JSON(500, map[string]interface{}{"error":err})
			return
		}
		log.Printf("QS I see this data for send message from f:\n %+v", message)

		var result Message
		if message.From != "" && message.To != "" && message.Body != "" {
			if message.To == "all" {
				peoples, _ := qs.GetPeoples(bson.M{})
				log.Printf("QSERV: will send [%v] to all %v peoples", message.Body, len(peoples))
				SendMessagesToPeoples(peoples, ntf, message.Body)
			} else if message.To == "all_team_members" {
				peoples, _ := qs.GetAllTeamMembers()
				log.Printf("QSERV: will send [%v] to all team members %v peoples", message.Body, len(peoples))
				SendMessagesToPeoples(peoples, ntf, message.Body)
			} else {
				team, _ := qs.GetTeamByName(message.To)
				if team == nil {
					man, _ := qs.GetManByUserId(message.To)
					if man != nil {
						log.Printf("QSERV: will send [%v] to %v", message.Body, man.UserId)
						go ntf.NotifyText(man.UserId, message.Body)
					}
				} else {
					peoples, _ := qs.GetMembersOfTeam(team.Name)
					log.Printf("QSERV: will send [%v] to team members of %v team to %v peoples", message.Body, team.Name, len(peoples))
					SendMessagesToPeoples(peoples, ntf, message.Body)
				}
			}
			result, err = qs.StoreMessage(message.From, message.To, message.Body, false)
			if err != nil {
				log.Printf("QSERV: error at storing message %v", err)
				render.JSON(200, map[string]interface{}{"ok":false})
				return
			}
			result.TimeFormatted = result.Time.Format(time.Stamp)

		} else {
			render.JSON(200, map[string]interface{}{"ok":false})
		}
		render.JSON(200, map[string]interface{}{"ok":true, "message":result})

	})

	r.Post("/messages", func(render render.Render, req *http.Request) {
		type NewMessagesReq struct {
			For   string `json:"m_for"`
			After int64 `json:"after"`
		}
		q := NewMessagesReq{}
		request_body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			render.JSON(500, map[string]interface{}{"ok":false, "detail":"can not read request body"})
			return
		}
		err = json.Unmarshal(request_body, &q)
		if err != nil {
			render.JSON(500, map[string]interface{}{"ok":false, "detail":fmt.Sprintf("can not unmarshal request body %v \n %s", err, request_body)})
			return
		}

		messages, err := qs.GetMessages(bson.M{"from":q.For, "time_stamp":bson.M{"$gt":q.After}})
		if err != nil {
			render.JSON(500, map[string]interface{}{"ok":false, "detail":fmt.Sprintf("error in db: %v", err)})
			return
		}

		for i, message := range messages {
			team, _ := qs.GetTeamByName(message.From)
			if team != nil {
				messages[i].From = team.Name
			} else {
				man, _ := qs.GetManByUserId(message.From)
				if man != nil {
					messages[i].From = man.Name
				}

			}
			messages[i].TimeFormatted = message.Time.Format(time.Stamp)
		}

		render.JSON(200, map[string]interface{}{"messages":messages, "next_":time.Now().Unix()})
	})

	r.Post("/contacts", func(render render.Render, req *http.Request) {
		type NewContactsReq struct {
			After int64 `json:"after"`
			Exist []string `json:"exist"`
		}
		cr := NewContactsReq{}
		request_body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			render.JSON(500, map[string]interface{}{"ok":false, "detail":"can not read request body"})
			return
		}
		err = json.Unmarshal(request_body, &cr)
		if err != nil {
			render.JSON(500, map[string]interface{}{"ok":false, "detail":fmt.Sprintf("can not unmarshal request body %v \n %s", err, request_body)})
			return
		}
		contacts, err := qs.GetContactsAfter(cr.After)
		if err != nil {
			render.JSON(500, map[string]interface{}{"ok":false, "detail": err})
			return
		}
		new_contacts := []Contact{}
		old_contacts := []Contact{}

		for _, contact := range contacts {
			if utils.InS(contact.ID, cr.Exist) {
				old_contacts = append(old_contacts, contact)
			} else {
				new_contacts = append(new_contacts, contact)
			}
		}
		render.JSON(200, map[string]interface{}{
			"ok":true,
			"new":new_contacts,
			"old":old_contacts,
			"next_":time.Now().Unix(),
		})

	})

	r.Get("/manage", func(render render.Render, req *http.Request) {
		configResult, err := qs.GetMessageConfiguration(config.Chat.CompanyId)
		if err != nil {
			log.Printf("QS E: Can not load quest configuration for %v, because: %v", config.Chat.CompanyId, err)
		}
		render.HTML(200, "manage", map[string]interface{}{"config":configResult})
	})

	r.Post("/manage", binding.Bind(QuestMessageConfiguration{}), func(qmc QuestMessageConfiguration, ren render.Render, req *http.Request) {
		if req.FormValue("to-winner-on") == "on" {
			qmc.EndMessageForWinnersActive = true
		}
		if req.FormValue("to-not-winner-on") == "on" {
			qmc.EndMessageForNotWinnersActive = true
		}
		if req.FormValue("to-all-on") == "on" {
			qmc.EndMessageForAllActive = true
		}
		log.Printf("QS Manage: %+v", qmc)
		qmc.CompanyId = config.Chat.CompanyId
		err := qs.SetMessageConfiguration(qmc, true)
		if err != nil {
			log.Printf("QS ERROR at update quest message configuration for [%v], because: %v", qmc.CompanyId, err)
		}
		ren.Redirect("/manage", 302)
	})

	r.Post("/start_quest", func(ren render.Render) {
		qmc, err := qs.GetMessageConfiguration(config.Chat.CompanyId)
		if err != nil {
			log.Printf("QS E: Can not load quest configuration for %v, because: %v", config.Chat.CompanyId, err)
			ren.JSON(500, map[string]interface{}{"ok":false, "detail":err})
			return
		}
		if qmc.Started == false {
			qs.SetQuestStarted(config.Chat.CompanyId, true)
			ren.JSON(200, map[string]interface{}{"ok":true})
		} else {
			ren.JSON(200, map[string]interface{}{"ok":false, "detail":"already started"})
		}
	})

	r.Post("/stop_quest", func(ren render.Render) {
		qmc, err := qs.GetMessageConfiguration(config.Chat.CompanyId)
		if err != nil {
			log.Printf("QS E: Can not load quest configuration for %v, because: %v", config.Chat.CompanyId, err)
			ren.JSON(500, map[string]interface{}{"ok":false, "detail":err})
			return
		}
		if qmc.Started == true {
			qs.SetQuestStarted(config.Chat.CompanyId, false)
			teams, err := qs.GetAllTeams()
			if err != nil {
				log.Printf("QS QE E: errror at getting teams %v", err)
			}
			if qmc.EndMessageForAllActive == true {
				for _, team := range teams {
					log.Printf("QS Will send message to team: %v from Klichat", team.Name)
					members, _ := qs.GetMembersOfTeam(team.Name)
					SendMessagesToPeoples(members, additionalNotifier, qmc.EndMessageForAll)
				}
			}
			if qmc.EndMessageForWinnersActive == true {
				for _, team := range teams {
					if team.Winner == true {
						log.Printf("QS Will send WIN message to team: %v from quest", team.Name)
						members, _ := qs.GetMembersOfTeam(team.Name)
						SendMessagesToPeoples(members, ntf, qmc.EndMessageForWinners)
					}
				}
			}
			if qmc.EndMessageForNotWinnersActive == true {
				for _, team := range teams {
					if team.Winner == false {
						log.Printf("QS Will send NOT WIN message to team: %v from quest", team.Name)
						members, _ := qs.GetMembersOfTeam(team.Name)
						SendMessagesToPeoples(members, ntf, qmc.EndMessageForNotWinners)
					}
				}
			}

			ren.JSON(200, map[string]interface{}{"ok":true})
		} else {
			ren.JSON(200, map[string]interface{}{"ok":false, "detail":"already stopped"})
		}

	})

	r.Get("/delete_chat/:between", func(params martini.Params, render render.Render, req *http.Request) {
		between := params["between"]
		qs.Messages.RemoveAll(bson.M{"$or":[]bson.M{bson.M{"from":between}, bson.M{"to":between}}})
		render.Redirect(fmt.Sprintf("/chat?with=%v", between))
	})

	r.Post("/send_messages_at_quest_end", func(render render.Render, req *http.Request) {
		type Messages struct {
			Text    string `json:"text"`
			Teams   []string `json:"teams"`
			Exclude bool `json:"exclude"`
		}
		messages := Messages{}
		data, err := ioutil.ReadAll(req.Body)
		if err != nil {
			log.Printf("QS QE E: errror at reading req body %v", err)
			render.JSON(500, map[string]interface{}{"error":err})
			return
		}
		err = json.Unmarshal(data, &messages)
		if err != nil {
			log.Printf("QS QE E: at unmarshal json messages %v", err)
			render.JSON(500, map[string]interface{}{"error":err})
			return
		}
		log.Printf("QS I see this data for send messages at quest end:\n %+v", messages)

		if messages.Exclude {
			teams, err := qs.GetAllTeams()
			if err != nil {
				log.Printf("QS QE E: errror at getting teams %v", err)
				render.JSON(500, map[string]interface{}{"error":err})
				return
			}
			for _, team := range teams {
				if !utils.InS(team.Name, messages.Teams) {
					log.Printf("QS Will send message to team: %v", team.Name)
					members, _ := qs.GetMembersOfTeam(team.Name)
					SendMessagesToPeoples(members, additionalNotifier, messages.Text)
				}
			}
			render.JSON(200, map[string]interface{}{"ok":true})
		} else {
			for _, team_name := range messages.Teams {
				members, err := qs.GetMembersOfTeam(team_name)
				if err != nil {
					log.Printf("QS QE E: errror at getting team members %v", err)
					continue
				}
				log.Printf("QS Will send message to team: %v", team_name)
				SendMessagesToPeoples(members, additionalNotifier, messages.Text)
			}
			render.JSON(200, map[string]interface{}{"ok":true})
		}

	})

	r.Post("/delete_all_keys", func(render render.Render, req *http.Request) {
		//1. Steps or keys:
		si, _ := qs.Steps.RemoveAll(bson.M{})
		//2 Peoples
		pi, _ := qs.Peoples.UpdateAll(bson.M{
			"$and":[]bson.M{
				bson.M{"$or":[]bson.M{
					bson.M{"is_passerby":false},
					bson.M{"is_passerby":bson.M{"$exists":false}},
				}},
				bson.M{"$or":[]bson.M{
					bson.M{"team_name":bson.M{"$exists":true}},
					bson.M{"team_sid":bson.M{"$exists":true}},
				}},
			},
		},
			bson.M{
				"$set":bson.M{"is_passerby":true},
				"$unset":bson.M{"team_name":"", "team_sid":""},
			})
		//3 teams and messages
		teams := []Team{}
		qs.Teams.Find(bson.M{}).All(&teams)
		tc := 0
		mc := 0
		for _, team := range teams {
			mri, _ := qs.Messages.RemoveAll(bson.M{
				"$or":[]bson.M{
					bson.M{"from":team.Name},
					bson.M{"to":team.Name},
				}})
			mc += mri.Removed

			qs.Teams.RemoveId(team.ID)
			tc += 1
		}
		render.JSON(200, map[string]interface{}{
			"ok":true,
			"steps_removed":si.Removed,
			"peoples_updated":pi.Updated,
			"teams_removed":tc,
			"messages_removed":mc,
		})
	})

	r.Post("/founded_keys", func(ren render.Render, req *http.Request) {
		type T struct {
			Name string `json:"team"`
		}
		t := T{}
		body, _ := ioutil.ReadAll(req.Body)
		json.Unmarshal(body, &t)
		steps, _ := qs.GetSteps(bson.M{"for_team":t.Name, "is_found":true})
		ren.JSON(200, map[string]interface{}{"keys":steps})
	})

	type FoundKey struct {
		Name        string `bson:"name" json:"name"`
		Found       bool `bson:"found" json:"found"`
		Id          string `json:"id"`
		Description string `json:"description"`
	}
	type TeamInfo struct {
		TeamName string `bson:"team_name" json:"team_name"`
		Keys     []FoundKey `json:"keys"`
		Steps    []Step `bson:"steps"`
	}

	r.Get("/info_page", func(ren render.Render, req *http.Request) {
		result := []TeamInfo{}
		err := qs.Steps.Pipe([]bson.M{
			bson.M{"$group":bson.M{
				"_id":"$for_team",
				"team_name":bson.M{"$first":"$for_team"},
				"steps":bson.M{"$push":bson.M{
					"_id":"$_id",
					"is_found":"$is_found",
					"next_key":"$next_key",
					"start_key":"$start_key",
					"description":"$description",
				}}}},
			bson.M{"$sort":bson.M{
				"team_name":1}},
		}).All(&result)
		if err != nil {
			log.Printf("QS Error at aggregate info page %v", err)
		}
		for ti, teamInfo := range result {
			steps := SortSteps(teamInfo.Steps)
			keys := []FoundKey{}
			for _, step := range steps {
				keys = append(keys, FoundKey{Name:step.StartKey, Found:step.IsFound, Id:step.ID.Hex(), Description:step.Description})
			}
			result[ti].Keys = keys
			result[ti].Steps = []Step{}
		}
		ren.HTML(200, "info_page", map[string]interface{}{"teams":result})
	})
	r.Post("/info_page/update", func(ren render.Render, req *http.Request) {
		found_keys, err := qs.GetSteps(bson.M{"is_found":true})
		if err != nil {
			ren.JSON(500, map[string]interface{}{"ok":false, "error":err.Error()})
		}
		type UpdateKeyResult struct {
			Id string `json:"id"`
		}
		result := []UpdateKeyResult{}
		for _, key := range found_keys {
			result = append(result, UpdateKeyResult{Id:key.ID.Hex()})
		}
		ren.JSON(200, map[string]interface{}{"ok":true, "foundKeys":result})
	})

	r.Post("/start_quest", func(ren render.Render, req *http.Request) {
		ren.JSON(200, map[string]interface{}{"ok":true})
	})
	r.Post("/stop_quest")


	//m.MapTo(r, (*martini.Routes)(nil))
	log.Printf("Will start web server for quest at: %v", config.WebPort)
	m.Action(r.Handle)
	m.RunOnAddr(config.WebPort)
}
