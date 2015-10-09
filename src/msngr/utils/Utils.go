package utils
import (

	"fmt"
	"math/rand"
	"reflect"

	"regexp"
	"strings"

	"os"
	"log"
)


func GenId() string {
	return fmt.Sprintf("%d", rand.Int63())
}

func CheckErr(e error) {
	if e != nil {
		panic(e)
	}
}

func ToMap(in interface{}, tag string) (map[string]interface{}, error) {
	out := make(map[string]interface{})

	v := reflect.ValueOf(in)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// we only accept structs
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("ToMap only accepts structs; got %T", v)
	}

	typ := v.Type()
	for i := 0; i < v.NumField(); i++ {
		// gets us a StructField
		fi := typ.Field(i)
		if tagv := fi.Tag.Get(tag); tagv != "" {
			out[tagv] = v.Field(i).Interface()
		}
	}
	return out, nil
}

func FirstOf(data ...interface{}) interface{} {
	for _, data_el := range data {
		if data_el != "" {
			return data_el
		}
	}
	return ""
}

func In(p int, a []int) bool {
	for _, v := range a {
		if p == v {
			return true
		}
	}
	return false
}

func InS(p string, a []string) bool {
	for _, v := range a {
		if p == v {
			return true
		}
	}
	return false
}

func Contains(container string, elements []string) bool {
	container_elements := regexp.MustCompile("[a-zA-Zа-яА-Я]+").FindAllString(container, -1)
	ce_map := make(map[string]bool)
	for _, ce_element := range container_elements {
		ce_map[strings.ToLower(ce_element)] = true
	}
	result := true
	for _, element := range elements {
		_, ok := ce_map[strings.ToLower(element)]
		result = result && ok
	}
	return result
}

func SaveToFile(what, fn string) {
	f, err := os.OpenFile(fn, os.O_APPEND | os.O_WRONLY | os.O_CREATE, 0600)
	if err != nil {
		log.Printf("ERROR when save to file in open file %v [%v]", fn, err)
	}

	defer f.Close()

	if _, err = f.WriteString(what); err != nil {
		log.Printf("ERROR when save to file in write to file %v [%v]", fn, err)
	}
}
