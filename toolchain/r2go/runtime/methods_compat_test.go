package runtime

import "testing"

func TestQuotedFunctionAssignment(t *testing.T) {
	value, _ := evaluate(t, `"afill<-" <- function(x) x + 1
`+"`afill<-`"+`(4)`)
	result, ok := value.(*DoubleVector)
	if !ok || len(result.Data) != 1 || result.Data[0] != 5 {
		t.Fatalf("unexpected quoted-assignment result: %T %v", value, value)
	}
}

func TestMethodsCompatibilityRegistration(t *testing.T) {
	value, _ := evaluate(t, `setClass("Point", slots=list(x="numeric", y="numeric"))
setGeneric("coord", def=function(object) 0)
setMethod("coord", signature("Point"), function(object) object@x)
c(isGeneric("coord"), isS4(new("Point", x=1, y=2)))`)
	logical, ok := value.(*LogicalVector)
	if !ok || len(logical.Data) != 2 || logical.Data[0] != True || logical.Data[1] != True {
		t.Fatalf("unexpected methods compatibility result: %T %v", value, value)
	}
}

func TestRlangEnvironmentAliases(t *testing.T) {
	value, _ := evaluate(t, `parent <- new.env()
child <- new_env(answer=42, parent_env=parent)
c(is.environment(child), child$answer)`)
	items, ok := value.(*DoubleVector)
	if !ok || len(items.Data) != 2 || items.Data[0] != 1 || items.Data[1] != 42 {
		t.Fatalf("unexpected new_env result: %T %v", value, value)
	}
}

func TestStandaloneNamespaceReferenceDoesNotForcePackageSymbol(t *testing.T) {
	value, _ := evaluate(t, `magrittr::`+"`%>%`"+``)
	if _, ok := value.(*Closure); !ok {
		t.Fatalf("expected namespace placeholder closure, got %T", value)
	}
}

func TestSymbolicAndReplacementMethodRegistration(t *testing.T) {
	value, _ := evaluate(t, `generic <- function(x) 0
setMethod(generic, signature(x="Point"), function(x) 1)
setReplaceMethod("slot", signature(x="Point"), function(x, value) x)
c(is.list(representation(x="numeric")), is.function(generic))`)
	logical, ok := value.(*LogicalVector)
	if !ok || len(logical.Data) != 2 || logical.Data[0] != True || logical.Data[1] != True {
		t.Fatalf("unexpected symbolic methods result: %T %v", value, value)
	}
}

func TestVectorConstructorDefaults(t *testing.T) {
	value, _ := evaluate(t, `list(vector(mode="numeric"), vector("character", 3))`)
	items, ok := value.(*List)
	if !ok || len(items.Data) != 2 {
		t.Fatalf("unexpected vector constructor result: %T %v", value, value)
	}
	if numeric, ok := items.Data[0].(*DoubleVector); !ok || len(numeric.Data) != 0 {
		t.Fatalf("unexpected numeric vector: %T %v", items.Data[0], items.Data[0])
	}
	if character, ok := items.Data[1].(*CharacterVector); !ok || len(character.Data) != 3 {
		t.Fatalf("unexpected character vector: %T %v", items.Data[1], items.Data[1])
	}
}

func TestPrototypeAndClassUnionCompatibility(t *testing.T) {
	value, _ := evaluate(t, `setClassUnion("RasterStackBrick", c("RasterStack", "RasterBrick"))
list(is.list(prototype(index=vector(mode="numeric"))), is.function(RasterStackBrick))`)
	items, ok := value.(*List)
	if !ok || len(items.Data) != 2 {
		t.Fatalf("unexpected class compatibility result: %T %v", value, value)
	}
}
