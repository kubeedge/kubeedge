package main

import (
	"github.com/go-chassis/go-archaius"
	"github.com/go-chassis/go-archaius/sources/file-source"
	"github.com/go-mesh/openlogging"
	"log"
)

func main() {
	err := archaius.Init(archaius.WithRequiredFiles([]string{"./dir", "f1.yaml"}))
	if err != nil {
		openlogging.GetLogger().Error("Error:" + err.Error())
	}
	log.Println(archaius.Get("age"))
	log.Println(archaius.Get("name"))
	log.Println(archaius.Get("c"))
	log.Println(archaius.Get("b"))
	err = archaius.AddFile("f2.yaml")
	if err != nil {
		log.Panicln(err)
	}
	log.Println(archaius.Get("age"))
	log.Println(archaius.Get("name"))

	err = archaius.AddFile("f3.yaml", archaius.WithFileHandler(filesource.UseFileNameAsKeyContentAsValue))
	if err != nil {
		log.Panicln(err)
	}
	log.Println(archaius.GetString("f3.yaml", ""))
}
