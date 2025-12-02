module github.com/medatechnology/simpleorm/example

go 1.23.2

toolchain go1.24.10

replace github.com/medatechnology/simpleorm => ../

require github.com/medatechnology/simpleorm v0.0.0-00010101000000-000000000000

require (
	github.com/lib/pq v1.10.9 // indirect
	github.com/medatechnology/goutil v0.0.3 // indirect
)
