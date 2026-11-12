package ucum

//go:generate go run ./internal/cmd/fetchucum -tag v2.2 -out internal/data/ucum-essence.xml
//go:generate go run ./internal/cmd/genucum -in internal/data/ucum-essence.xml -out default_registry.go

// DefaultEngine is an [Engine] built with default options from
// [DefaultRegistry], the definitions compiled into this package.
var DefaultEngine = mustNew(DefaultRegistry)

// mustNew builds an engine from definitions that were validated when they
// were generated, so failure indicates a defect in this package.
func mustNew(registry *Registry) *Engine {
	engine, err := New(registry)
	if err != nil {
		panic(err)
	}
	return engine
}
