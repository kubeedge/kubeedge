package utils

import (
	"os/exec"
	"time"
)

func MakeMapperImages(makeMapperProject, getModbusCode, buildModbusMapperProject, makeMapperImage string) error {
	// build mapper project
	cmd := exec.Command("sh", "-c", makeMapperProject)
	if err := PrintCmdOutput(cmd); err != nil {
		return err
	}

	cmd = exec.Command("sh", "-c", getModbusCode)
	if err := PrintCmdOutput(cmd); err != nil {
		return err
	}

	cmd = exec.Command("sh", "-c", buildModbusMapperProject)
	if err := PrintCmdOutput(cmd); err != nil {
		return err
	}

	// check images exist
	Infof("begin build mapper images")
	cmd = exec.Command("sh", "-c", makeMapperImage)
	if err := PrintCmdOutput(cmd); err != nil {
		return err
	}
	return nil
}

func CheckMapperImage(checkMapperImage string) error {
	cmd := exec.Command("sh", "-c", checkMapperImage)
	if err := PrintCmdOutput(cmd); err != nil {
		return err
	}
	return nil
}

// run mapper
func RunMapper(runMapper, checkMapperRun string) error {
	Infof("run mapper image on docker")
	time.Sleep(1 * time.Second)
	cmd := exec.Command("sh", "-c", runMapper)
	if err := PrintCmdOutput(cmd); err != nil {
		return err
	}

	time.Sleep(5 * time.Second)

	Infof("check whether mapper container run successfully")
	cmd = exec.Command("sh", "-c", checkMapperRun)
	if err := PrintCmdOutput(cmd); err != nil {
		return err
	}
	return nil
}

// stop mapper container
func RemoveMapperContainer(deleteMapperContainer string) error {
	Infof("stop mapper container running")
	cmd := exec.Command("sh", "-c", deleteMapperContainer)
	if err := PrintCmdOutput(cmd); err != nil {
		return err
	}
	return nil
}

// delete mapper image
func RemoveMapperImage(deleteMapperImage string) error {
	cmd := exec.Command("sh", "-c", deleteMapperImage)
	if err := PrintCmdOutput(cmd); err != nil {
		return err
	}
	return nil
}

// delete mapper project
func RemoveMapperProject(deleteMapperProject, deleteModbusCode string) error {
	cmd := exec.Command("sh", "-c", deleteMapperProject)
	if err := PrintCmdOutput(cmd); err != nil {
		return err
	}
	cmd = exec.Command("sh", "-c", deleteModbusCode)
	if err := PrintCmdOutput(cmd); err != nil {
		return err
	}
	return nil
}
