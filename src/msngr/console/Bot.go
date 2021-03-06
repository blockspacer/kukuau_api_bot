package console

import (
	c "msngr/configuration"
	d "msngr/db"
	m "msngr"
	s "msngr/structs"
	n "msngr/notify"
	"strings"
	"log"
	"regexp"
	"msngr/quests"
	"fmt"
	"msngr/voting"
)

const (
	ME = "me"
)

var key_reg = regexp.MustCompile("^\\#[\\w\\dа-яА-Я]+\\-?(?P<team>[\\w\\da-zа-я]+)?")

func GetCommands() *[]s.OutCommand {
	return &[]s.OutCommand{
		s.OutCommand{
			Title:    "Информация",
			Action:   "information",
			Position: 1,
		},
	}
}

type ConsoleRequestProcessor struct {

}

func (crp *ConsoleRequestProcessor)ProcessRequest(in *s.InPkg) *s.RequestResult {
	result := s.RequestResult{Commands:GetCommands()}
	return &result
}

type ConsoleInformationProcessor struct {
	Information string
}

func (cip ConsoleInformationProcessor) ProcessMessage(in *s.InPkg) *s.MessageResult {
	result := s.MessageResult{Type:"chat", Body:cip.Information}
	return &result
}

type ConsoleMessageProcessor struct {
	d.MainDb
	QuestStorage *quests.QuestStorage
}

func (cmp ConsoleMessageProcessor) ProcessMessage(in *s.InPkg) *s.MessageResult {
	body := in.Message.Body
	userData := in.UserData
	if body != nil && userData != nil {
		u, _ := cmp.Users.GetUserById(in.From)
		if u == nil {
			cmp.Users.AddUser(in.From, userData.Name, userData.Phone, userData.Email)
		} else {
			cmp.Users.UpdateUserData(in.From, userData.Name, userData.Phone, userData.Email)
		}
		r_body := *body
		cmp.Messages.StoreMessage(in.From, ME, r_body, in.Message.ID)
		r_body = strings.ToLower(strings.TrimSpace(r_body))
		if key_reg.MatchString(r_body) {
			log.Printf("CC: Here is key: %v", r_body)
			step, err := cmp.QuestStorage.GetStepByStartKey(r_body)
			if step != nil {
				cmp.Users.SetUserState(in.From, "last_marker", r_body)
				return &s.MessageResult{Type:"chat", Body:step.Description}
			}
			if step == nil && err == nil {

				keys, err := cmp.QuestStorage.GetAllSteps()
				key_s := []string{}
				for _, k := range keys {
					key_s = append(key_s, k.StartKey)
				}
				if err == nil {
					return &s.MessageResult{Type:"chat", Body:fmt.Sprintf("Попробуте другие ключи! Я знаю такие: %+v.", strings.Join(key_s, " "))}
				}
			}
		}
		return &s.MessageResult{Type:"chat", Body:"", IsDeferred:true}
	} else {
		return &s.MessageResult{Type:"chat", Body:"Нет данных для сообщения или данных пользователя"}
	}
}

func FormConsoleBotContext(conf c.Configuration, db *d.MainDb, configStore *d.ConfigurationStorage) *m.BotContext {
	result := m.BotContext{}
	result.RequestProcessors = map[string]s.RequestCommandProcessor{
		"commands":&ConsoleRequestProcessor{},
	}

	companyId := config.Console.Chat.CompanyId
	commandsGenerator := func(in *s.InPkg) (*[]s.OutCommand, error) {
		return GetCommands(), nil
	}
	result.MessageProcessors = map[string]s.MessageCommandProcessor{
		"":m.NewFuncTextBodyProcessor(db, commandsGenerator, companyId, nil),
		"information":m.NewUpdatableInformationProcessor(configStore, commandsGenerator, companyId),
	}

	notifier := n.NewNotifier(conf.Main.CallbackAddr, conf.Console.Chat.Key, db)
	qs := quests.NewQuestStorage(conf.Main.Database.ConnString, conf.Main.Database.Name)

	vdh, _ := voting.NewVotingHandler(conf.Main.Database.ConnString, conf.Main.Database.Name)
	go Run(conf.Console.WebPort, db, qs, vdh, notifier, conf)

	return &result
}

