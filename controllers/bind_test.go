package controllers

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"aunefyren/poenskelisten/models"
	"aunefyren/poenskelisten/utilities"

	"github.com/gin-gonic/gin"
)

// boundRequestTypes maps every request type the handlers bind, spelled as in
// the source, to its reflect.Type, so TestEveryBoundRequestIsSanitized can
// check its tags. Source code can't be turned into a type at test time, hence
// the list; the test fails on a bound type missing from it, so the list can't
// fall behind.
var boundRequestTypes = map[string]reflect.Type{
	"TokenRequest":                               reflect.TypeOf(TokenRequest{}),
	"resetRequest":                               reflect.TypeOf(resetRequest{}),
	"models.GroupAddWishlistsRequest":            reflect.TypeOf(models.GroupAddWishlistsRequest{}),
	"models.GroupCreationRequest":                reflect.TypeOf(models.GroupCreationRequest{}),
	"models.GroupMembershipCreationRequest":      reflect.TypeOf(models.GroupMembershipCreationRequest{}),
	"models.GroupMembershipRemovalRequest":       reflect.TypeOf(models.GroupMembershipRemovalRequest{}),
	"models.GroupUpdateRequest":                  reflect.TypeOf(models.GroupUpdateRequest{}),
	"models.MFAActivateRequest":                  reflect.TypeOf(models.MFAActivateRequest{}),
	"models.MFADisableRequest":                   reflect.TypeOf(models.MFADisableRequest{}),
	"models.MFAValidateRequest":                  reflect.TypeOf(models.MFAValidateRequest{}),
	"models.NewsCreationRequest":                 reflect.TypeOf(models.NewsCreationRequest{}),
	"models.NewsUpdateRequest":                   reflect.TypeOf(models.NewsUpdateRequest{}),
	"models.OAuthRegisterRequest":                reflect.TypeOf(models.OAuthRegisterRequest{}),
	"models.ServerSettingsRequest":               reflect.TypeOf(models.ServerSettingsRequest{}),
	"models.UpdateCurrencyRequest":               reflect.TypeOf(models.UpdateCurrencyRequest{}),
	"models.UserCreationRequest":                 reflect.TypeOf(models.UserCreationRequest{}),
	"models.UserUpdatePasswordRequest":           reflect.TypeOf(models.UserUpdatePasswordRequest{}),
	"models.UserUpdateRequest":                   reflect.TypeOf(models.UserUpdateRequest{}),
	"models.WishClaimCreationRequest":            reflect.TypeOf(models.WishClaimCreationRequest{}),
	"models.WishCreationRequest":                 reflect.TypeOf(models.WishCreationRequest{}),
	"models.WishUpdateRequest":                   reflect.TypeOf(models.WishUpdateRequest{}),
	"models.WishlistCollaboratorCreationRequest": reflect.TypeOf(models.WishlistCollaboratorCreationRequest{}),
	"models.WishlistCollaboratorDeletionRequest": reflect.TypeOf(models.WishlistCollaboratorDeletionRequest{}),
	"models.WishlistCreationRequest":             reflect.TypeOf(models.WishlistCreationRequest{}),
	"models.WishlistMembershipCreationRequest":   reflect.TypeOf(models.WishlistMembershipCreationRequest{}),
	"models.WishlistMembershipDeletionRequest":   reflect.TypeOf(models.WishlistMembershipDeletionRequest{}),
	"models.WishlistUpdateRequest":               reflect.TypeOf(models.WishlistUpdateRequest{}),
}

// directBindAllowed lists the files that may call gin's binding methods
// themselves: bindJSON's own file, and the dynamic client registration
// endpoint, whose error format RFC 7591 dictates (it calls
// utilities.SanitizeRequest itself, which this test also checks).
var directBindAllowed = map[string]bool{"bind.go": true, "oauth_register.go": true}

var ginBindMethods = map[string]bool{
	"Bind": true, "BindJSON": true, "BindWith": true,
	"ShouldBind": true, "ShouldBindJSON": true, "ShouldBindWith": true, "ShouldBindBodyWith": true,
}

// Every request body is cleaned by its struct's sanitize tags, which only
// happens if the handler binds through bindJSON (or calls SanitizeRequest)
// and every string field of the bound struct carries a tag.
func TestEveryBoundRequestIsSanitized(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("no controller sources found: %v", err)
	}

	fset := token.NewFileSet()
	seen := map[string]bool{}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			t.Fatal(err)
		}

		sanitizedHere := map[string]bool{}
		directlyBound := map[string]token.Pos{}
		for _, decl := range parsed.Decls {
			function, ok := decl.(*ast.FuncDecl)
			if !ok || function.Body == nil {
				continue
			}
			ast.Inspect(function.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				var target ast.Expr
				switch fun := call.Fun.(type) {
				case *ast.Ident:
					if fun.Name == "bindJSON" && len(call.Args) == 2 {
						target = call.Args[1]
					}
				case *ast.SelectorExpr:
					if ginBindMethods[fun.Sel.Name] {
						if !directBindAllowed[file] {
							t.Errorf("%s: %s called directly; bind through bindJSON so the sanitize tags apply", fset.Position(call.Pos()), fun.Sel.Name)
						}
						if file != "bind.go" && len(call.Args) > 0 {
							target = call.Args[0]
							if name, ok := addressedIdent(target); ok {
								directlyBound[name] = call.Pos()
							}
						}
					} else if isSelector(fun, "utilities", "SanitizeRequest") && len(call.Args) == 1 {
						if name, ok := addressedIdent(call.Args[0]); ok {
							sanitizedHere[name] = true
						}
					}
				}
				if target == nil {
					return true
				}

				typeName, ok := boundTypeName(function, target)
				if !ok {
					t.Errorf("%s: can't tell which type is bound here; bind a local variable declared with an explicit type", fset.Position(call.Pos()))
					return true
				}
				seen[typeName] = true
				if _, known := boundRequestTypes[typeName]; !known {
					t.Errorf("%s: %s isn't in boundRequestTypes; add it so its sanitize tags are checked", fset.Position(call.Pos()), typeName)
				}
				return true
			})
		}
		for name, pos := range directlyBound {
			if !sanitizedHere[name] {
				t.Errorf("%s: %s is bound directly but never passed to utilities.SanitizeRequest", fset.Position(pos), name)
			}
		}
	}

	for typeName, requestType := range boundRequestTypes {
		if !seen[typeName] {
			t.Errorf("boundRequestTypes lists %s, but no handler binds it; remove it", typeName)
			continue
		}
		for _, problem := range utilities.UntaggedStringFields(requestType) {
			t.Errorf("%s", problem)
		}
	}
}

func isSelector(expr *ast.SelectorExpr, pkg string, name string) bool {
	ident, ok := expr.X.(*ast.Ident)
	return ok && ident.Name == pkg && expr.Sel.Name == name
}

// addressedIdent returns v for the expression &v.
func addressedIdent(expr ast.Expr) (string, bool) {
	unary, ok := expr.(*ast.UnaryExpr)
	if !ok || unary.Op != token.AND {
		return "", false
	}
	ident, ok := unary.X.(*ast.Ident)
	if !ok {
		return "", false
	}
	return ident.Name, true
}

// boundTypeName finds the declared type of the variable in &v by looking for
// `var v T` or `v := T{}` in the enclosing function.
func boundTypeName(function *ast.FuncDecl, target ast.Expr) (string, bool) {
	name, ok := addressedIdent(target)
	if !ok {
		return "", false
	}

	var typeExpr ast.Expr
	ast.Inspect(function.Body, func(node ast.Node) bool {
		switch declaration := node.(type) {
		case *ast.ValueSpec:
			for _, ident := range declaration.Names {
				if ident.Name == name && declaration.Type != nil {
					typeExpr = declaration.Type
				}
			}
		case *ast.AssignStmt:
			if declaration.Tok != token.DEFINE {
				return true
			}
			for i, lhs := range declaration.Lhs {
				ident, ok := lhs.(*ast.Ident)
				if !ok || ident.Name != name || i >= len(declaration.Rhs) {
					continue
				}
				if literal, ok := declaration.Rhs[i].(*ast.CompositeLit); ok {
					typeExpr = literal.Type
				}
			}
		}
		return true
	})

	switch typed := typeExpr.(type) {
	case *ast.Ident:
		return typed.Name, true
	case *ast.SelectorExpr:
		if pkg, ok := typed.X.(*ast.Ident); ok {
			return pkg.Name + "." + typed.Sel.Name, true
		}
	}
	return "", false
}

func bindJSONRequest(body string, request any) (bool, int, map[string]interface{}) {
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("POST", "/", strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ok := bindJSON(ctx, request)

	var parsed map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &parsed)
	return ok, w.Code, parsed
}

func TestBindJSONSanitizes(t *testing.T) {
	var request models.UserCreationRequest
	ok, _, _ := bindJSONRequest(`{"first_name":"  ada ","last_name":" lovelace","email":" Ada@Example.COM ","password":" Pass word1 "}`, &request)
	if !ok {
		t.Fatal("bindJSON failed on a valid body")
	}
	want := models.UserCreationRequest{FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com", Password: " Pass word1 "}
	if request != want {
		t.Errorf("request = %+v, want %+v", request, want)
	}
}

func TestBindJSONMalformedBody(t *testing.T) {
	var request models.UserCreationRequest
	ok, code, body := bindJSONRequest("not json", &request)
	if ok || code != 400 || body["error"] != "Failed to parse request." {
		t.Errorf("bindJSON = (%v, %d, %v), want false, 400 and a parse error", ok, code, body)
	}
}

func TestBindJSONRejectsUnsafeText(t *testing.T) {
	var request models.WishCreationRequest
	ok, code, body := bindJSONRequest(`{"name":"<script>alert(1)</script>"}`, &request)
	_, requirements := utilities.ValidateTextCharacters("<")
	if ok || code != 400 || body["error"] != requirements {
		t.Errorf("bindJSON = (%v, %d, %v), want false, 400 and the text requirements", ok, code, body)
	}
}

func TestBindJSONBadTagIsInternalError(t *testing.T) {
	var request struct {
		Name string `json:"name" sanitize:"bogus"`
	}
	ok, code, body := bindJSONRequest(`{"name":"x"}`, &request)
	if ok || code != 500 || body["error"] != "Failed to process request." {
		t.Errorf("bindJSON = (%v, %d, %v), want false, 500", ok, code, body)
	}
}
