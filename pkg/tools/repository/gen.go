package repository

//go:generate genny -in=generic_repository.go -out=nse_repository.gen.go gen "Generic=registry.NetworkServiceEndpoint"
//go:generate genny -in=generic_repository.go -out=nsm_repository.gen.go gen "Generic=registry.NetworkServiceManager"
//go:generate genny -in=generic_repository.go -out=ns_repository.gen.go gen "Generic=registry.NetworkService"
