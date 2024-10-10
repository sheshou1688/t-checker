package checker

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"t-checker/gurl"
	"time"

	"github.com/360EntSecGroup-Skylar/excelize"
	gsjson "github.com/bitly/go-simplejson"
	"github.com/panjf2000/ants"
)

const (
	Api   = "https://appmall.ciotour.com/electronic-code/ticket-query"
	MCode = "dwr1Op9C3C5qVSng3ZWSFA"
)

var cur, _ = os.Getwd()

type Checker struct {
	Date          string
	Threads       int
	ThreadsPool   *ants.Pool
	Tasks         []Task
	SuccessResult []*Result
	FailResult    []*Result
	Wg            *sync.WaitGroup
}

type Task struct {
	Name       string
	Credential string
	Date       string
}

type Result struct {
	VisitorName   string
	Credential    string
	TicketNumber  string
	CrowdTypeName string
	TourDate      string
	StartDate     string
	EndDate       string
	Tickets       []*Ticket
}

type Ticket struct {
	SkuName         string
	ChildStatusName string `json:"childStatusName"`
	OrderNo         string `json:"orderNo"`
	OrderSourceName string `json:"orderSourceName"`
}

var Ckr *Checker

// https://appmall.ciotour.com/electronic-code/ticket-query?m_code=dwr1Op9C3C5qVSng3ZWSFA&visitorName=%E9%BB%84%E8%BE%BE%E9%B9%8F&credential=330327198708070231&tourDate=2024-10-10
func NewChecker(date string) *Checker {
	Ckr = &Checker{Date: date, Wg: new(sync.WaitGroup)}
	return Ckr
}

func (cr *Checker) Init(max int) {
	fmt.Println("开始初始化...")
	var pool, err = ants.NewPool(max, ants.WithNonblocking(false), ants.WithPreAlloc(true))
	if err != nil {
		fmt.Printf("初始化池失败: %s \n", err.Error())
		time.Sleep(time.Second * 3)
		log.Fatal(err)
	}

	cr.ThreadsPool = pool

	cr.loadTaskList()
	fmt.Println("初始化完毕!!!")
}

func (cr *Checker) loadTaskList() {
	fmt.Println("开始加载待检查资料...")

	file, err := os.Open(cur + "/check.txt")
	if err != nil {
		fmt.Printf("读取待检查文件失败: %s \n", err.Error())
		time.Sleep(time.Second * 3)
		log.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	var list []string
	for scanner.Scan() {
		list = append(list, scanner.Text())
	}
	if scanner.Err() != nil {
		fmt.Printf("读取待检查文件行失败: %s \n", err.Error())
		time.Sleep(time.Second * 3)
		log.Fatal(err)
	}

	for _, v := range list {
		dataArr := strings.Split(strings.TrimSpace(v), " ")
		if len(dataArr) < 2 {
			fmt.Printf("跳过不符合格式的行: %s \n", v)
			continue
		}
		cr.Tasks = append(cr.Tasks, Task{Name: dataArr[0], Credential: dataArr[1], Date: cr.Date})
	}

	if len(cr.Tasks) <= 0 {
		fmt.Printf("没有读取到任何资料,结束!\n")
		time.Sleep(time.Second * 3)
		log.Fatal(err)
	}

	fmt.Printf("检查文件加载完毕! 名单条数: %d ;第一个资料为: %s-%s \n", len(cr.Tasks), cr.Tasks[0].Name, cr.Tasks[0].Credential)
}

func (cr *Checker) Run() (err error) {
	rand.Seed(time.Now().UnixNano())
	for _, v := range cr.Tasks {
		cr.Wg.Add(1)
		time.Sleep(time.Millisecond * (100 + time.Duration(rand.Intn(200))))
		var task = v
		cr.ThreadsPool.Submit(task.check)
	}

	cr.Wg.Wait()
	fmt.Println(cr.SuccessResult, "===")
	fmt.Println(cr.FailResult, "===")
	return
}

func (tk *Task) check() {
	var err error
	var result = &Result{VisitorName: tk.Name, Credential: tk.Credential, TourDate: tk.Date}
	defer func() {
		Ckr.Wg.Done()
		if err == nil {
			Ckr.SuccessResult = append(Ckr.SuccessResult, result)
		} else {
			Ckr.FailResult = append(Ckr.FailResult, result)
		}
	}()
	fmt.Printf("开始查询 %s - %s 的票... \n", tk.Name, tk.Credential)
	rsp, err := gurl.New(http.MethodGet, Api+fmt.Sprintf("?m_code=%s&visitorName=%s&credential=%s&tourDate=%s", MCode, url.QueryEscape(tk.Name), tk.Credential, tk.Date)).Set(gurl.Option{Timeout: time.Second * 10}).Do()
	if err != nil {
		fmt.Printf("查询 %s 的票出错: %s \n", tk.Name, err.Error())
		time.Sleep(time.Second)
		rsp, err = gurl.New(http.MethodGet, Api+fmt.Sprintf("?m_code=%s&visitorName=%s&credential=%s&tourDate=%s", MCode, url.QueryEscape(tk.Name), tk.Credential, tk.Date)).Set(gurl.Option{Timeout: time.Second * 10}).Do()
		if err != nil {
			fmt.Printf("查询 %s 的票出错: %s \n", tk.Name, err.Error())
			return
		}
	}

	js, err := gsjson.NewJson(rsp)
	if err != nil {
		fmt.Printf("查询 %s 的票出错 (NewJson): %s ; 原数据: %s \n", tk.Name, err.Error(), string(rsp))
		return
	}

	result.TicketNumber, err = js.Get("data").Get("ticketNumber").String()
	if err != nil {
		fmt.Printf("查询 %s 的票出错 (ticketNumber): %s ;原数据: %s \n", tk.Name, err.Error(), string(rsp))
	}

	result.CrowdTypeName, err = js.Get("data").Get("crowdTypeName").String()
	if err != nil {
		fmt.Printf("查询 %s 的票出错 (crowdTypeName): %s ;原数据: %s \n", tk.Name, err.Error(), string(rsp))
	}

	result.StartDate, err = js.Get("data").Get("startDate").String()
	if err != nil {
		fmt.Printf("查询 %s 的票出错 (startDate): %s ;原数据: %s \n", tk.Name, err.Error(), string(rsp))
	}

	result.StartDate, err = js.Get("data").Get("endDate").String()
	if err != nil {
		fmt.Printf("查询 %s 的票出错 (endDate): %s ;原数据: %s \n", tk.Name, err.Error(), string(rsp))
	}

	ticketsArr := js.Get("data").Get("electronicCodeProductProviderOutBOS")

	for i := 0; i < 10; i++ {
		ticket := &Ticket{}
		_, exist := ticketsArr.GetIndex(i).CheckGet("skuName")
		if !exist {
			break
		}
		ticket.SkuName, err = ticketsArr.GetIndex(i).Get("skuName").String()
		if err != nil {
			fmt.Printf("查询 %s 的票出错 (skuName): %s \n", tk.Name, err.Error())
			return
		}
		ticket.ChildStatusName, err = ticketsArr.GetIndex(i).Get("childStatusName").String()
		if err != nil {
			fmt.Printf("查询 %s 的票出错 (childStatusName): %s \n", tk.Name, err.Error())
			return
		}
		ticket.OrderNo, _ = ticketsArr.GetIndex(i).Get("orderNo").String()
		ticket.OrderSourceName, _ = ticketsArr.GetIndex(i).Get("orderSourceName").String()
		result.Tickets = append(result.Tickets, ticket)
	}

}

func (cr *Checker) writeResult() (err error) {
	var sheet = "查询结果表"
	var xlsx = excelize.NewFile()
	indexSheet := xlsx.NewSheet(sheet)
	xlsx.AutoFilter(sheet, "C1", "P1", "")
	xlsx.DeleteSheet("Sheet1")
	// 设置宽度
	xlsx.SetColWidth(sheet, "A", "U", 25)
	// 设置行高
	xlsx.SetRowHeight(sheet, 1, 30)
	// 设置冻结窗口
	xlsx.SetPanes(sheet, `{"freeze":true,"split":false,"x_split":1,"y_split":1,"top_left_cell":"B2","active_pane":"topRight","panes":[{"sqref":"","active_cell":"","pane":"topRight"}]}`)
	// 设置对齐方式
	style, err := xlsx.NewStyle(`{"alignment":{"horizontal":"center","Vertical":"center"},"font":{"bold":true,"family":"微软雅黑"},"fill":{"type":"pattern","pattern":1}}`)
	if err != nil {
		return
	}
	xlsx.SetCellStyle(sheet, "A1", "U1", style)

	var headers = map[string]string{
		"A1": "姓名",
		"B1": "身份证",
		"C1": "优惠",
		"D1": "预定日期",
		"E1": "开始日期",
		"F1": "结束日期",
		"G1": "项目1",
		"H1": "使用状态",
		"I1": "项目2",
		"J1": "使用状态",
		"K1": "项目3",
		"L1": "使用状态",
		"M1": "项目4",
		"N1": "使用状态",
		"O1": "项目5",
		"P1": "使用状态",
		"Q1": "项目6",
		"R1": "使用状态",
		"S1": "项目7",
		"T1": "使用状态",
		"U1": "项目8",
		"V1": "使用状态",
		"W1": "项目9",
		"X1": "使用状态",
		"Y1": "项目10",
		"Z1": "使用状态",
	}
	for k, v := range headers {
		xlsx.SetCellStr(sheet, k, v)
	}

	// 主体写入数据

	var count int
	if len(cr.SuccessResult) > 0 {
		for k, v := range cr.SuccessResult {
			lint := strconv.Itoa(k + 2)
			// 设置行高
			xlsx.SetRowHeight(sheet, k+2, 25)
			// 设置对齐方式
			style1, err := xlsx.NewStyle(`{"alignment":{"horizontal":"center","Vertical":"center"}}`)
			if err != nil {
				fmt.Printf("写入结果失败;err: %s \n", err.Error())
				return err
			}
			xlsx.SetCellStyle(sheet, "A"+lint, "U"+lint, style1)
			var datas = map[string]string{
				"A" + lint: v.VisitorName,
				"B" + lint: v.Credential,
				"C" + lint: v.CrowdTypeName,
				"D" + lint: v.TourDate,
				"E" + lint: v.StartDate,
				"F" + lint: v.EndDate,
				"G" + lint: "",
				"H" + lint: "",
				"I" + lint: "",
				"J" + lint: "",
				"K" + lint: "",
				"L" + lint: "",
				"M" + lint: "",
				"N" + lint: "",
				"O" + lint: "",
				"P" + lint: "",
				"Q" + lint: "",
				"R" + lint: "",
				"S" + lint: "",
				"T" + lint: "",
				"U" + lint: "",
				"V" + lint: "",
				"W" + lint: "",
				"X" + lint: "",
				"Y" + lint: "",
				"Z" + lint: "",
			}
			var idxKey = []string{"G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z"}
			for key, val := range v.Tickets {
				idx := key * 2
				datas[idxKey[idx]+lint] = val.SkuName
				datas[idxKey[idx+1]+lint] = val.SkuName
			}
			for key, value := range datas {
				xlsx.SetCellValue(sheet, key, value)
			}
			count++
		}
		lints := strconv.Itoa(int(count) + 2)
		xlsx.SetRowHeight(sheet, int(count)+2, 60)
		// xlsx.MergeCell(sheet, "C"+lints, "U"+lints)
		xlsx.SetCellStyle(sheet, "A"+lints, "B"+lints, style)

		var datas = map[string]string{
			"A" + lints: "统计",
			"F" + lints: fmt.Sprintf("成功个数：%d", count),
		}
		for key, value := range datas {
			xlsx.SetCellValue(sheet, key, value)
		}
	}
	lints := strconv.Itoa(int(count) + 2)
	xlsx.SetRowHeight(sheet, int(count)+2, 60)
	// xlsx.MergeCell(sheet, "C"+lints, "U"+lints)
	xlsx.SetCellStyle(sheet, "A"+lints, "B"+lints, style)
	xlsx.SetActiveSheet(indexSheet)
	// ctype := mime.TypeByExtension(".xlsx")
	xlsx.SaveAs(cur + "/" + time.Now().Format("20060102150405") + "-" + Ckr.Date)

	if len(cr.SuccessResult) > 0 {

	}
	return
}

func (cr *Checker) writeFailResult() {
	file, err := os.Create(cur + "/" + "fail.txt")
	if err != nil {
		return
	}
}
