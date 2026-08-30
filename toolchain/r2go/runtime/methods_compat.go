package runtime

import (
	"fmt"

	"github.com/tarekwasfy01/RTDAL---R-Transpiled-Data-Abstraction-Library/toolchain/r2go/syntax"
)

func compatibilityArgument(ctx *Context, args []syntax.Argument, env *Environment, positional int, names ...string) (Value, bool, error) {
	for _, name := range names {
		for _, arg := range args {
			if arg.Name == name {
				value, err := ctx.Eval(arg.Value, env)
				return value, true, err
			}
		}
	}
	if positional >= 0 && positional < len(args) {
		value, err := ctx.Eval(args[positional].Value, env)
		return value, true, err
	}
	return NullValue, false, nil
}

func compatibilityName(ctx *Context, args []syntax.Argument, env *Environment, positional int, names ...string) (string, bool, error) {
	candidates := make([]syntax.Argument, 0, 1)
	for _, name := range names {
		for _, arg := range args {
			if arg.Name == name {
				candidates = append(candidates, arg)
				break
			}
		}
	}
	if len(candidates) == 0 && positional >= 0 && positional < len(args) {
		candidates = append(candidates, args[positional])
	}
	if len(candidates) == 0 {
		return "", false, nil
	}
	switch expression := candidates[0].Value.(type) {
	case *syntax.Symbol:
		return expression.Name, true, nil
	case *syntax.Literal:
		if expression.Kind == syntax.StringLiteral {
			return expression.Text, true, nil
		}
	}
	value, err := ctx.Eval(candidates[0].Value, env)
	if err != nil {
		return "", true, err
	}
	name, err := compatibilityText(value)
	return name, true, err
}

func compatibilityText(value Value) (string, error) {
	if text, ok := value.(*CharacterVector); ok && len(text.Data) != 0 {
		return text.Data[0], nil
	}
	if language, ok := value.(*Language); ok {
		return language.String(), nil
	}
	return "", fmt.Errorf("expected a non-empty character value")
}

func (c *Context) methodsCompatibilityBuiltin(name string, args []syntax.Argument, env *Environment) (Value, error) {
	switch name {
	case "representation", "prototype":
		values := make([]Value, 0, len(args))
		names := make([]string, 0, len(args))
		for _, arg := range args {
			value, err := c.Eval(arg.Value, env)
			if err != nil {
				return nil, err
			}
			values = append(values, value)
			names = append(names, arg.Name)
		}
		return &List{Data: values, Names: names}, nil

	case "signature":
		values := make([]string, 0, len(args))
		names := make([]string, 0, len(args))
		for _, arg := range args {
			value, err := c.Eval(arg.Value, env)
			if err != nil {
				return nil, err
			}
			text, err := compatibilityText(value)
			if err != nil {
				return nil, err
			}
			values = append(values, text)
			names = append(names, arg.Name)
		}
		result := &CharacterVector{Data: values}
		if len(names) != 0 {
			result.Attr = map[string]Value{"names": &CharacterVector{Data: names}}
		}
		return result, nil

	case "globalVariables", "setLoadAction", "loadModule":
		// Static Pure-Go builds do not need R namespace/native-module
		// registration side effects. Keep package initialisation successful.
		return NullValue, nil

	case "setClass", "setOldClass", "setClassUnion":
		classValue, ok, err := compatibilityArgument(c, args, env, 0, "Class", "class")
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("%s requires a class name", name)
		}
		className, err := compatibilityText(classValue)
		if err != nil {
			return nil, err
		}
		generator := NewNativeFunction(c.Global, []NativeParameter{{Name: "..."}}, func(ctx *Context, frame *Environment) (Value, error) {
			value, err := frame.Get(ctx, "...")
			if err != nil {
				return nil, err
			}
			dots, ok := value.(*List)
			if !ok {
				dots = &List{}
			}
			return &List{
				Data:  append([]Value(nil), dots.Data...),
				Names: append([]string(nil), dots.Names...),
				Attr: map[string]Value{
					"class": &CharacterVector{Data: []string{className}},
					"S4":    &LogicalVector{Data: []Logical{True}},
				},
			}, nil
		})
		c.Global.Set(className, generator)
		return generator, nil

	case "setGeneric", "setGroupGeneric":
		genericValue, ok, err := compatibilityArgument(c, args, env, 0, "name", "f")
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("%s requires a generic name", name)
		}
		genericName, err := compatibilityText(genericValue)
		if err != nil {
			return nil, err
		}
		definition, found, err := compatibilityArgument(c, args, env, 1, "def", "definition")
		if err != nil {
			return nil, err
		}
		if found {
			if _, function := definition.(*Closure); !function {
				return nil, fmt.Errorf("generic definition for %s is not a function", genericName)
			}
			c.Global.Set(genericName, definition)
			return definition, nil
		}
		if current, lookupErr := c.Global.Get(c, genericName); lookupErr == nil {
			return current, nil
		}
		return NullValue, nil

	case "isGeneric":
		genericValue, ok, err := compatibilityArgument(c, args, env, 0, "f")
		if err != nil || !ok {
			return boolValue(false), err
		}
		genericName, err := compatibilityText(genericValue)
		if err != nil {
			return nil, err
		}
		value, lookupErr := c.Global.Get(c, genericName)
		if lookupErr != nil {
			return boolValue(false), nil
		}
		_, function := value.(*Closure)
		return boolValue(function), nil

	case "setMethod", "setReplaceMethod":
		genericName, ok, err := compatibilityName(c, args, env, 0, "f")
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("%s requires a generic name", name)
		}
		if name == "setReplaceMethod" && (len(genericName) < 2 || genericName[len(genericName)-2:] != "<-") {
			genericName += "<-"
		}
		signatureValue, _, err := compatibilityArgument(c, args, env, 1, "signature")
		if err != nil {
			return nil, err
		}
		definition, found, err := compatibilityArgument(c, args, env, 2, "definition", "def")
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, fmt.Errorf("setMethod requires a method definition")
		}
		methodName := genericName + ".default"
		if signatureText, signatureErr := compatibilityText(signatureValue); signatureErr == nil && signatureText != "" {
			methodName = genericName + "." + signatureText
		}
		c.Global.Set(methodName, definition)
		if _, lookupErr := c.Global.Get(c, genericName); lookupErr != nil {
			c.Global.Set(genericName, definition)
		}
		return definition, nil

	case "setAs":
		fromValue, fromOK, err := compatibilityArgument(c, args, env, 0, "from")
		if err != nil || !fromOK {
			return nil, fmt.Errorf("setAs requires a source class")
		}
		toValue, toOK, err := compatibilityArgument(c, args, env, 1, "to")
		if err != nil || !toOK {
			return nil, fmt.Errorf("setAs requires a target class")
		}
		from, err := compatibilityText(fromValue)
		if err != nil {
			return nil, err
		}
		to, err := compatibilityText(toValue)
		if err != nil {
			return nil, err
		}
		definition, found, err := compatibilityArgument(c, args, env, 2, "def")
		if err != nil {
			return nil, err
		}
		if found {
			c.Global.Set("coerce."+from+"."+to, definition)
			return definition, nil
		}
		return NullValue, nil

	case "setValidity":
		return NullValue, nil

	case "getClass":
		classValue, ok, err := compatibilityArgument(c, args, env, 0, "Class")
		if err != nil || !ok {
			return nil, fmt.Errorf("getClass requires a class name")
		}
		className, err := compatibilityText(classValue)
		if err != nil {
			return nil, err
		}
		if generator, lookupErr := c.Global.Get(c, className); lookupErr == nil {
			return generator, nil
		}
		return &CharacterVector{Data: []string{className}}, nil

	case "new":
		classValue, ok, err := compatibilityArgument(c, args, env, 0, "Class")
		if err != nil || !ok {
			return nil, fmt.Errorf("new requires a class name")
		}
		className, err := compatibilityText(classValue)
		if err != nil {
			return nil, err
		}
		values := make([]Value, 0, len(args)-1)
		names := make([]string, 0, len(args)-1)
		for index, arg := range args {
			if (index == 0 && arg.Name == "") || arg.Name == "Class" {
				continue
			}
			value, err := c.Eval(arg.Value, env)
			if err != nil {
				return nil, err
			}
			values = append(values, value)
			names = append(names, arg.Name)
		}
		return &List{Data: values, Names: names, Attr: map[string]Value{
			"class": &CharacterVector{Data: []string{className}},
			"S4":    &LogicalVector{Data: []Logical{True}},
		}}, nil
	}
	return nil, fmt.Errorf("unsupported methods compatibility builtin %s", name)
}

func (c *Context) vectorConstructor(args []syntax.Argument, env *Environment) (Value, error) {
	mode := "logical"
	if value, ok, err := compatibilityArgument(c, args, env, 0, "mode"); err != nil {
		return nil, err
	} else if ok {
		mode, err = compatibilityText(value)
		if err != nil {
			return nil, err
		}
	}
	length := 0
	if value, ok, err := compatibilityArgument(c, args, env, 1, "length"); err != nil {
		return nil, err
	} else if ok {
		length, err = scalarInt(value)
		if err != nil || length < 0 {
			return nil, fmt.Errorf("invalid vector length")
		}
	}
	switch mode {
	case "logical":
		return &LogicalVector{Data: make([]Logical, length)}, nil
	case "integer":
		return &IntegerVector{Data: make([]int64, length)}, nil
	case "double", "numeric":
		return &DoubleVector{Data: make([]float64, length)}, nil
	case "complex":
		return &ComplexVector{Data: make([]complex128, length)}, nil
	case "character":
		return &CharacterVector{Data: make([]string, length)}, nil
	case "raw":
		return &RawVector{Data: make([]byte, length)}, nil
	case "list", "expression", "pairlist", "any":
		return &List{Data: make([]Value, length)}, nil
	default:
		return nil, fmt.Errorf("invalid vector mode %q", mode)
	}
}
func (c *Context) namespaceReference(args []syntax.Argument, env *Environment) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("namespace reference expects package and name")
	}
	var name string
	switch target := args[1].Value.(type) {
	case *syntax.Symbol:
		name = target.Name
	case *syntax.Literal:
		name = target.Text
	default:
		return nil, fmt.Errorf("invalid namespace member")
	}
	if value, err := env.Get(c, name); err == nil {
		return value, nil
	}
	return NewNativeFunction(c.Global, []NativeParameter{{Name: "..."}}, func(*Context, *Environment) (Value, error) {
		return nil, fmt.Errorf("external namespace function %s is unavailable in the Pure-Go runtime", name)
	}), nil
}

func (c *Context) environmentCompatibilityBuiltin(name string, args []syntax.Argument, env *Environment) (Value, error) {
	switch name {
	case "emptyenv":
		return NewEnvironment(nil), nil
	case "baseenv", "globalenv", "asNamespace":
		return c.Global, nil
	case "parent.frame":
		if env.Parent != nil {
			return env.Parent, nil
		}
		return c.Global, nil
	case "parent.env":
		value, ok, err := compatibilityArgument(c, args, env, 0)
		if err != nil {
			return nil, err
		}
		if !ok {
			value = env
		}
		target, ok := value.(*Environment)
		if !ok {
			return nil, fmt.Errorf("argument is not an environment")
		}
		if target.Parent == nil {
			return NewEnvironment(nil), nil
		}
		return target.Parent, nil
	}
	return nil, fmt.Errorf("unsupported environment builtin %s", name)
}

func isS4CompatibilityObject(value Value) bool {
	flag, ok := Attributes(value)["S4"].(*LogicalVector)
	return ok && len(flag.Data) != 0 && flag.Data[0] == True
}
