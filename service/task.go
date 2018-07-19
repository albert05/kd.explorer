package service

import (
	"fmt"
	"time"
	"kd.explorer/model"
	"encoding/json"
	"strings"
	"kd.explorer/common"
	"kd.explorer/util/https"
	"kd.explorer/util/dates"
	"strconv"
	"kd.explorer/util/mysql"
)

type TaskResponse struct {
	Code int ``
	Message string ``
}

const ExchangeURL = "https://deposit.koudailc.com/user-order-form/convert"

func GoRunTask(taskList []mysql.MapModel) {
	ch := make(chan string)
	for _, task := range taskList {
		go runT(task, ch)
	}

	for range taskList {
		<-ch
	}

	close(ch)
}

func runT(task mysql.MapModel, ch chan<- string) {
	taskId := task.GetAttrInt("id")
	fmt.Println(fmt.Sprintf("taskID %d start work", taskId))
	userKey := task.GetAttrString("user_key")

	cookie, err := LoginK(userKey)
	if err != nil {
		ch <- "login failed"
		return
	}

	var code Code
	code.setCookie(cookie)
	code.Refresh()
	code.RandFileName()
	code.CreateImage()

	fmt.Println(cookie, code.getFileName())

	model.UpdateTask(taskId, map[string]string {
		"img_url": code.getFileName(),
	})

	timePoint := task.GetAttrFloat("time_point")
	imgCode := wait(timePoint, taskId)

	pid := task.GetAttrString("product_id")
	prizeNumber := task.GetAttrString("prize_number")

	params := map[string]string{
		"id": pid,
		"imgcode": imgCode,
		"prize_number": prizeNumber,
	}

	body, err := https.Post(ExchangeURL, params, cookie)
	if err != nil {
		fmt.Println(err)
		ch <- err.Error()
		return
	}

	var result TaskResponse
	json.Unmarshal(body, &result)

	status := 3
	msg := ""
	if https.HttpSUCCESS != result.Code {
		status = 2
		msg = result.Message
	}
	model.UpdateTask(taskId, map[string]string {
		"status": strconv.Itoa(status),
		"result": msg,
	})

	fmt.Println(userKey + " -- " + string(body))
	ch <- "success"
}

func wait(timePoint float64, taskId int) string {
	currTime := dates.TimeInt2float(dates.CurrentMicro())
	fmt.Println(currTime, timePoint)

	var imgCode string

	for currTime < timePoint {
		time.Sleep(common.DefaultSleepTIME)

		if imgCode == "" {
			task := model.FindTask(taskId)

			imgCode = strings.Trim(task.GetAttrString("code"), " ")
		}

		currTime = dates.TimeInt2float(dates.CurrentMicro())
	}

	return imgCode
}
