package store

// Catalog holds the default list of store applications loaded from embedded app packages.
var Catalog []StoreApp

func init() {
	apps, err := LoadEmbeddedCatalog()
	if err == nil && len(apps) > 0 {
		Catalog = apps
	}
}
