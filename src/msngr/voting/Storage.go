package voting

import (
	d "msngr/db"
	"gopkg.in/mgo.v2"
	"errors"
	"gopkg.in/mgo.v2/bson"
	"fmt"
	"msngr/utils"
	"reflect"
)

type Voter struct {
	UserName string `bson:"user_name"`
	Role     string `bson:"role,omitempty"`
}

func (v Voter) String() string {
	if v.Role != "" {
		return fmt.Sprintf("%v (%v)", v.UserName, v.Role)
	}
	return v.UserName
}

type VoteObject struct {
	VoteCount int `bson:"vote_count"`
	Voters    []Voter `bson:"voters"`
}

func (vo VoteObject) String() string {
	return fmt.Sprintf("\n\tcount: %v, users:%+v", vo.VoteCount, vo.Voters)
}
func (vo VoteObject) ContainUserName(userName string) bool {
	for _, fVouter := range vo.Voters {
		if fVouter.UserName == userName {
			return true
		}
	}
	return false
}

type CompanyModel struct {
	VoteInfo    VoteObject `bson:"vote"`
	ID          bson.ObjectId `bson:"_id,omitempty"`
	Name        string `bson:"name"`
	City        string `bson:"city"`
	Service     string `bson:"service"`
	Description string `bson:"description"`
}

func (cm CompanyModel) Get(fieldBsonName string) string {
	v := reflect.ValueOf(cm)
	typ := v.Type()
	for i := 0; i < v.NumField(); i++ {
		// gets us a StructField
		fi := typ.Field(i)
		if tagv := fi.Tag.Get("bson"); tagv == fieldBsonName {
			return v.Field(i).String()
		}
	}
	return ""
}
func (cm CompanyModel) String() string {
	return fmt.Sprintf("\n-------------------\nCompany: [%v] \nName:%v\nCity:%v\nDescription:%v\nVotes:%+v\n-------------------\n",
		cm.ID, cm.Name, cm.City, cm.Description, cm.VoteInfo)
}

type VotingDataHandler struct {
	d.DbHelper
	Companies *mgo.Collection
}

func (vdh *VotingDataHandler) ensureIndexes() {
	companiesCollection := vdh.Session.DB(vdh.DbName).C("vote_companies")
	companiesCollection.EnsureIndex(mgo.Index{
		Key:        []string{"name", "city", "service"},
		Background: true,
		DropDups:   true,
		Unique:    true,
	})
	companiesCollection.EnsureIndex(mgo.Index{
		Key:        []string{"vote.voters.user_name"},
		Background: true,
		DropDups:   true,
		Unique:    false,
	})
	companiesCollection.EnsureIndex(mgo.Index{
		Key:        []string{"name"},
		Background: true,
		DropDups:   true,
		Unique:    false,
	})
	companiesCollection.EnsureIndex(mgo.Index{
		Key:        []string{"city"},
		Background: true,
		DropDups:   true,
		Unique:    false,
	})
	companiesCollection.EnsureIndex(mgo.Index{
		Key:        []string{"service"},
		Background: true,
		DropDups:   true,
		Unique:    false,
	})
	vdh.Companies = companiesCollection
}

func NewVotingHandler(conn, dbName string) (*VotingDataHandler, error) {
	dbh := d.NewDbHelper(conn, dbName)
	if dbh.Check() {
		result := VotingDataHandler{DbHelper:*dbh}
		result.ensureIndexes()
		return &result, nil
	}
	return nil, errors.New("Can not connect to db, try it next time")
}

func (vdh *VotingDataHandler) ConsiderCompany(name, city, service, description, userName, userRole string) error {
	found := CompanyModel{}
	err := vdh.Companies.Find(bson.M{"name":name, "city":city, "service":service}).One(&found)
	if err == mgo.ErrNotFound {
		err = vdh.Companies.Insert(CompanyModel{
			Name:name,
			City:city,
			Description:description,
			Service:service,
			VoteInfo:VoteObject{
				Voters:[]Voter{
					Voter{UserName:userName, Role: userRole}},
				VoteCount:1,
			},
		})
		return err
	} else {
		if found.VoteInfo.ContainUserName(userName) {
			return errors.New("This user already vote this")
		}
		voter := Voter{UserName:userName, Role:userRole}
		vdh.Companies.UpdateId(found.ID, bson.M{
			"$inc":bson.M{"vote.vote_count": 1},
			"$addToSet":bson.M{"vote.voters":voter},
		})
	}
	return nil
}

func (vdh *VotingDataHandler) GetCompanies(q bson.M) ([]CompanyModel, error) {
	result := []CompanyModel{}
	err := vdh.Companies.Find(q).All(&result)
	return result, err
}

func (vdh *VotingDataHandler) TextFoundByCompanyField(q, field string) ([]string, error) {
	result := []string{}
	qResult := []CompanyModel{}
	if !utils.InS(field, []string{"name", "city", "service"}) {
		return result, errors.New("field invalid")
	}
	err := vdh.Companies.Find(bson.M{field:bson.RegEx{fmt.Sprintf(".*%v.*", q), ""}}).All(&qResult)
	if err != nil && err != mgo.ErrNotFound {
		return result, err
	} else if err == mgo.ErrNotFound {
		return result, nil
	}
	for _, cm := range qResult {
		result = append(result, cm.Get(field))
	}
	return result, nil
}

func (vdh *VotingDataHandler) GetUserVotes(username string) ([]CompanyModel, error) {
	result := []CompanyModel{}
	err := vdh.Companies.Find(bson.M{"vote.voters.user_name":username}).All(&result)
	return result, err
}