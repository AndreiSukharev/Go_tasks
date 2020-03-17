package main

import (
	"fmt"
	"reflect"
)

func i2s(data interface{}, out interface{}) error {
	//fmt.Printf("%T", data)
	valPtr := reflect.ValueOf(out)
	if valPtr.Kind() != reflect.Ptr {
		return fmt.Errorf("data is not ptr")
	} else {
		valPtr = valPtr.Elem()
	}
	//fmt.Println(valPtr.Kind())
	switch valPtr.Kind() {
	case reflect.Struct:
		d, ok := data.(map[string]interface{})
		if !ok {
			return fmt.Errorf("failed convert to map[string]interface{}")
		}
		for i := 0; i < valPtr.NumField(); i++ {
			nameField := valPtr.Type().Field(i).Name
			val, ok := d[nameField]
			if !ok {
				return fmt.Errorf("field not found: %s", val)
			}
			if err := i2s(val, valPtr.Field(i).Addr().Interface()); err != nil {
				return fmt.Errorf("failed to process struct field %s: %s", val, err)
			}
		}
	case reflect.Slice:
		arr, ok := data.([]interface{})
		if !ok {
			return fmt.Errorf("failed convert to []interface{}")
		}
		for i := 0; i < len(arr); i++ {
			newVal := reflect.New(valPtr.Type().Elem())
			if err := i2s(arr[i], newVal.Interface()); err != nil {
				return fmt.Errorf("failed to process slice element %d: %s", i, err)
			}
			valPtr.Set(reflect.Append(valPtr, newVal.Elem()))
		}
	case reflect.String:
		val, ok := data.(string)

		if !ok {
			return fmt.Errorf("string err")
		}
		valPtr.SetString(val)
	case reflect.Int:
		val, ok := data.(float64)
		if !ok {
			return fmt.Errorf("num err")
		}
		valPtr.SetInt(int64(val))
	case reflect.Bool:
		val, ok := data.(bool)
		if !ok {
			return fmt.Errorf("bool err")
		}
		valPtr.SetBool(val)
	}
	return nil
}
