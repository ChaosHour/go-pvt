# go-pvt
A Go cli that will find all MySQL stored procedures, functions, views and events with definer for a given DB.


## Usage
```Go


go-pvt on  main [?] via 🐹 v1.21.1 
❯ go run . -h                        
Usage of /var/folders/pb/6_zqqsb11w9f16t02h4gyf800000gn/T/go-build2453480786/b001/exe/go-pvt:
  -d string
        Database Name
  -s string
        Source Host
  -show
        Show Databases


go-pvt on  main [?] via 🐹 v1.21.1 
❯ go run . -s primary -show          
Connected to primary (primary): ✔

Databases:
mysql
information_schema
performance_schema
sys
char_test_db

go-pvt on  main [?] via 🐹 v1.21.1 
❯ go run . -s primary -d char_test_db
Connected to primary (primary): ✔

Total: 4

Objects:

NAME                    TYPE            DEFINER        
my_proc                 (PROCEDURE)     root@localhost
my_view                 (VIEW)          root@localhost
set_default_salary      (TRIGGER)       root@localhost
my_event                (EVENT)         root@localhost

```

## Screenshots

<img src="screenshots/Screenshot 2023-09-14 at 10.14.40 AM.png" width="585" height="369" />





## How to install
```Go

go install github.com/ChaosHour/go-pvt


To build:

go build -o go-pvt

FreeBSD:
env GOOS=freebsd GOARCH=amd64 go build .

On Mac:
env GOOS=darwin GOARCH=amd64 go build .

Linux:
env GOOS=linux GOARCH=amd64 go build .
```