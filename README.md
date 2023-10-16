# pkgdmp - Go code analysis tool

`pkgdmp` is a simple command-line tool for analyzing directories containing Go code. It provides an overview of functions, structs, methods, function types, and interfaces defined in the packages within those directories.

**Features:**

- Customizable output options
- Entity filtering using patterns
- Multiple syntax highlighting themes
- JSON output generation

## Usage

```console
user@example:~$ pkgdmp -help

USAGE:

  pkgdmp [FLAGS] DIRECTORY [DIRECTORY2] ...

FLAGS:

  -exclude string
        exclude entities with names matching regular expression [$PKGDMP_EXCLUDE]
  -full-doc
        include full doc comments instead of synopsis [$PKGDMP_FULL_DOC]
  -json
        output as JSON [$PKGDMP_JSON]
  -match string
        only include entities with names matching regular expression [$PKGDMP_MATCH]
  -no-doc
        exclude doc comments [$PKGDMP_NO_DOC]
  -no-env
        skip loading of configuration from 'PKGDMP_*' environment variables
  -no-func-types
        exclude function types [$PKGDMP_NO_FUNC_TYPES]
  -no-funcs
        exclude functions [$PKGDMP_NO_FUNCS]
  -no-highlight
        skip source code highlighting [$PKGDMP_NO_HIGHLIGHT]
  -no-interfaces
        exclude interfaces [$PKGDMP_NO_INTERFACES]
  -no-structs
        exclude structs [$PKGDMP_NO_STRUCTS]
  -theme string
        syntax highlighting theme to use - see https://xyproto.github.io/splash/docs/ [$PKGDMP_THEME] (default "swapoff")
  -unexported
        include unexported entities [$PKGDMP_UNEXPORTED]
  -version
        print version information and exit
```

## Examples

Analyze the contents of the `myproject` directory and display all exported entities:

```console
user@example:~$ pkgdmp myproject
package mypackage

// MyFunctionType is a function type that takes two integers and returns a
// boolean.
type MyFunctionType func(int, int) bool

// MyStruct is a struct with exported and unexported fields.
type MyStruct struct {
        ExportedField int // exported field.
}

// MyMethod is a method associated with MyStruct.
func (s MyStruct) MyMethod()

. . .
```

Analyze the `myproject` directory, excluding entities matching pattern and displaying full documentation comments in JSON format:

```console
user@example:~$ pkgdmp -exclude "^My.*Function$" -full-doc -json myproject
[
  {
    "name": "mypackage",
    "funcs": [
      {
        "name": "NewMyStruct",
        "synopsis": "NewMyStruct is an example constructor function for [MyStruct]\n",
        "params": [
          {
            "type": {
              "name": "int"
            },
            "names": [
              "n"
            ]
          }
        ],
        "results": [
          {
            "type": {
              "name": "MyStruct",
              "prefix": "*"
            },
            "names": null
          },
. . .
```

## Installation

Grab a pre-compiled version from the [release page](https://github.com/michenriksen/pkgdmp/releases) or install the latest version with Go:

```console
user@example:~$ go install github.com/michenriksen/pkgdmp@latest
```
