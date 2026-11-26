package main

import (
	_ "LSSKiosk/routers"
	beego "github.com/beego/beego/v2/server/web"
)

func main() {
	beego.Run()
}

