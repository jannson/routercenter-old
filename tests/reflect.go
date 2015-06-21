package main

import (
	"errors"
	"fmt"
	"reflect"
)

type R1 struct {
	i int
}

func (r *R1) Test1(b int) (int, error) {
	fmt.Println(b)
	return 444, nil
}

func InvokeFun(fn interface{}, params ...interface{}) (result []reflect.Value, err error) {
	f := reflect.ValueOf(fn)
	if len(params) != f.Type().NumIn() {
		err = errors.New("The number of params is not adapted.")
		return
	}
	in := make([]reflect.Value, len(params))
	for k, param := range params {
		in[k] = reflect.ValueOf(param)
	}
	result = f.Call(in)
	return
}

func (r1 *R1) Call(name string, params ...interface{}) (result []reflect.Value, err error) {
	//f := reflect.ValueOf(m[name])
	//f := reflect.ValueOf(r1).MethodByName(name)
	fmt.Println(name)
	f := reflect.ValueOf(r1.Test1)
	if len(params) != f.Type().NumIn() {
		err = errors.New("The number of params is not adapted.")
		return
	}
	in := make([]reflect.Value, len(params))
	for k, param := range params {
		in[k] = reflect.ValueOf(param)
	}
	result = f.Call(in)
	return
}

func Invoke(any interface{}, name string, args ...interface{}) {
	inputs := make([]reflect.Value, len(args))
	for i, _ := range args {
		inputs[i] = reflect.ValueOf(args[i])
	}
	reflect.ValueOf(any).MethodByName(name).Call(inputs)
}

//http://stackoverflow.com/questions/18091562/how-to-get-underlying-value-from-a-reflect-value-in-golang/29668838#29668838
//switch val.Kind() {
//case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
//    m[typeField.Name] = strconv.FormatInt(val.Int(), 10)
//case reflect.String:
//    m[typeField.Name] = val.String()
//}

func main() {
	r := new(R1)
	//v := Invoke(r, "Test1", 5)
	v, _ := InvokeFun(r.Test1, 8)
	fmt.Println(v[0].Int())
}
