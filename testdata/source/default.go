package mypackage

import (
	"errors"
	"fmt"
)

// MyExportedType is an exported custom type.
type MyExportedType int

// myUnexportedType is an unexported custom type.
type myUnexportedType string

// MyFunctionType is a function type that takes two integers and returns a
// boolean.
type MyFunctionType func(int, int) bool

// MyFunction is an example function that takes two integers as input and
// returns a boolean result. It compares the values of the input integers
// and returns true if they are equal, indicating a successful comparison.
// Otherwise, it returns false to indicate that the integers are not equal.
//
// This function serves as a simple equality checker and is often used to
// demonstrate the usage of function types in Go.
//
// Example usage:
//
//	result := MyFunction(5, 5) // result will be true
//	result := MyFunction(10, 20) // result will be false
//
// Parameters:
//
//	a: The first integer to compare.
//	b: The second integer to compare.
//
// Returns:
//
//	true if the integers are equal, false otherwise.
func MyFunction(a, b int) bool {
	return a == b
}

// MyOtherFunction is an exported function that does not match [MyFunctionType].
func MyOtherFunction(s string, cb func(string) bool) bool {
	return cb(s)
}

// MyThirdFunction returns a function type.
func MyThirdFunction() MyFunctionType {
	return MyFunction
}

// MyStruct is a struct with exported and unexported fields.
type MyStruct struct {
	ExportedField                      int    // exported field.
	unexportedField                    string // unexported field.
	unexportedField1, unexportedField2 int    // unexported shorthand fields.
}

// NewMyStruct is an example constructor function for [MyStruct]
func NewMyStruct(n int) (*MyStruct, error) {
	if n < 0 {
		return nil, errors.New("n must be a positive integer")
	}

	return &MyStruct{ExportedField: n}, nil
}

// MyMethod is a method associated with MyStruct.
func (s MyStruct) MyMethod() {
	fmt.Println("MyMethod called")
}

// myUnexportedMethod is an example unexported method.
func (s MyStruct) myUnexportedMethod(a, b string) string {
	return fmt.Sprintf("%s: %s", a, b)
}

// MyInterface is an interface with a single method.
type MyInterface interface {
	MyMethod() error
}

// myUnexportedInterface is an unexported interface.
type myUnexportedInterface interface {
	AnotherMethod(string, int, MyFunctionType) (n int, err error)
}

// myUnexportedFunction is an unexported function.
func myUnexportedFunction(a string, b int) string {
	return fmt.Sprint("%s: %d\n", a, b)
}
