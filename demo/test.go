package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	log "github.com/jeanphorn/log4go"
	"github.com/larspensjo/config"
	"go8583/byteutil"
	"go8583/netutil"
	"go8583/up8583"
	"io"
	"os"
	"strconv"
	"time"
)

var (
	conFile        = flag.String("configfile", "/config.ini", "config file")
	Server  string = "127.0.0.1"
	Port    int    = 5050

	up *up8583.Up8583

	cardvailddata string
	cardholdsn    string
	cardbin       string

	//全局变量
	records [][]string
)

func checkErr(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
	}
}

func configUP() {

	//获取当前路径
	file, _ := os.Getwd()
	cfg, err := config.ReadDefault(file + *conFile)
	checkErr(err)
	//获取配置文件中的配置项
	Server, err = cfg.String("SERVERCONFIG", "Server")
	Port, err = cfg.Int("SERVERCONFIG", "Port")
	log.Info("Server:%s\n", Server)
	log.Info("Port:%d\n", Port)

	up8583.ManNum, _ = cfg.String("UPCONFIG", "ManNum")
	up8583.PosNum, _ = cfg.String("UPCONFIG", "PosNum")
	up8583.MainKey, _ = cfg.String("UPCONFIG", "MainKey")
	up8583.TPDU, _ = cfg.String("UPCONFIG", "TPDU")

	up8583.RecSn, _ = cfg.Int("RECCONFIG", "RecSn") //记录流水

	up = up8583.NewUp8583()
}

/*
签到处理过程
*/
func QdProc() error {
	log.Info("connect:server=%s,port=%d\n", Server, Port)
	conn, err := netutil.Connect(Server, Port)
	if err == nil {
		log.Info("connect ok!")
	} else {
		log.Info("connect failed!")
		return err
	}
	defer netutil.DisConnect(conn)
	up.Frame8583QD()
	up.Ea.PrintFields(up.Ea.Field_S)
	_, err = netutil.TxData(conn, up.Ea.Txbuf)
	if err == nil {
		log.Info("send ok!")
	}
	rxbuf := make([]byte, 1024)
	rxlen := 0
	rxlen, err = netutil.RxData(conn, rxbuf)
	if err == nil {
		log.Info("recv ok!len=%d\n", rxlen)
		err = up.Ans8583QD(rxbuf, rxlen)
		if err == nil {
			log.Info("签到成功")
		} else {
			log.Info("签到失败")
			log.Info(err)
		}
	}
	return err

}

/*
银联二维码交易
*/
func QrcodeProc() error {

	log.Info("connect:server=%s,port=%d\n", Server, Port)
	conn, err := netutil.Connect(Server, Port)
	if err == nil {
		log.Info("connect ok!")
	} else {
		log.Info("connect failed!")
		return err
	}
	defer netutil.DisConnect(conn)
	up.Frame8583Qrcode("6220485073630469936", 1)
	up.Ea.PrintFields(up.Ea.Field_S)
	_, err = netutil.TxData(conn, up.Ea.Txbuf)
	if err == nil {
		log.Info("send ok!")
	}
	rxbuf := make([]byte, 1024)
	rxlen := 0
	rxlen, err = netutil.RxData(conn, rxbuf)
	if err == nil {
		log.Info("recv ok!len=%d\n", rxlen)
		up.Ea.Ans8583Fields(rxbuf, rxlen)
		up.Ea.PrintFields(up.Ea.Field_R)
	}
	return err
}

/*
银联电子现金交易
*/
//获取55域 IC卡数据域
/*
9F26 08
9F27 01
9F10
9F37 04
9F36 02
95 05
9A 03
9C 01
9F02 06
5F2A 02
82 02
9F1A 02
9F03 06
9F33 03
9F1E 08
84
9F09 02
9F41 04
9F34 03
9F35 01
9F63 10
9F74 06
8A 02
*/
func getField55(filed55, date string) []byte {
	//st := "0000000049c6f6c135bbf05b250000000001900000000000000000000000015601560051406e677c00056013071e0103900000010a010000001580b69553a06013827504000384486d25032240e0e1c80f250300021030313034353530300000000000006000064543433030310a60138275040000"
	//st := "00000000483f7ee5314c8565c300000000019000000000000000000000000156015600611d181c7c00009a13071e0103900000010a010000010145a1f302076013827504000669696d27042240e0e1c80f270400021030313034353530300000000000006000064543433030310a60138275040000"
	st := filed55
	//money := "190" //1.9元
	//cardbin := "601382750400038"
	l := 0
	s := ""

	l += 5 * 2

	s += "9F2608"
	s += st[l : l+8*2]
	l += 8 * 2

	s += "9F0206"
	s += st[l : l+6*2]
	l += 6 * 2

	s += "9F0306"
	s += st[l : l+6*2]
	l += 6 * 2

	s += "9505"
	s += st[l : l+5*2]
	l += 5 * 2

	s += "9F1A02"
	s += st[l : l+2*2]
	l += 2 * 2

	s += "5F2A02"
	s += st[l : l+2*2]
	l += 2 * 2

	s += "9C01"
	s += st[l : l+1*2]
	l += 1 * 2

	s += "9F3704"
	s += st[l : l+4*2]
	l += 4 * 2

	s += "8202"
	s += st[l : l+2*2]
	l += 2 * 2

	s += "9F3602"
	s += st[l : l+2*2]
	l += 2 * 2

	s += "9F10"
	tmplen, _ := strconv.ParseInt(st[l:l+1*2], 16, 16)
	s += st[l : l+1*2]
	l += 1 * 2
	s += st[l : l+int(tmplen)*2]
	l += 32 * 2

	s += "9F2701"
	s += st[l : l+1*2]
	l += 1 * 2

	s += "9F3303"
	s += st[l : l+3*2]
	l += 3 * 2

	l += 1 * 2 //可选位元表
	//卡有效期
	cardvailddata = st[l : l+2*2]
	l += 2 * 2

	fmt.Printf("cardvailddata:%s\n", cardvailddata)
	//卡片序列号
	cardholdsn = st[l : l+2*2]
	l += 2 * 2
	fmt.Printf("cardholdsn:%s\n", cardholdsn)
	//卡产品标识
	s += "9F63"
	tmplen, _ = strconv.ParseInt(st[l:l+1*2], 16, 16)
	s += st[l : l+1*2]
	l += 1 * 2
	s += st[l : l+int(tmplen)*2]
	l += 16 * 2
	//电子现金发卡行授权码9F74
	s += "9F74"
	tmplen, _ = strconv.ParseInt(st[l:l+1*2], 16, 16)
	s += st[l : l+1*2]
	l += 1 * 2
	s += st[l : l+int(tmplen)*2]

	//fmt.Println(st)
	s += "9A03"
	//s += "181205"
	s += date
	s += "9F1E083030303030303030"
	s += "9f0902008c"
	s += "9f3403000000"
	s += "9f350136"
	s += "8A025931"
	s += "9F4104"
	s += fmt.Sprintf("%08d", up8583.RecSn)
	fmt.Println(s)
	return byteutil.HexStringToBytes(s)
}

func UpCashProc(carbin string, money int, date string, field55data string) error {

	fmt.Printf("connect:server=%s,port=%d\n", Server, Port)
	conn, err := netutil.Connect(Server, Port)
	if err == nil {
		fmt.Println("connect ok!")
	} else {
		fmt.Println("connect failed!")
		return err
	}
	defer netutil.DisConnect(conn)

	field55 := getField55(field55data, date)
	//cardbin = "601382750400038"
	//cardbin = "601382750400074"
	up.Frame8583UpCash(carbin, money, cardvailddata, cardholdsn, field55)
	up.Ea.PrintFields(up.Ea.Field_S)
	log.Info(byteutil.BytesToHexString(up.Ea.Txbuf))
	_, err = netutil.TxData(conn, up.Ea.Txbuf)
	if err == nil {
		log.Info("send ok!")
	}
	rxbuf := make([]byte, 1024)
	rxlen := 0
	rxlen, err = netutil.RxData(conn, rxbuf)
	if err == nil {
		log.Info("recv ok!len=%d\n", rxlen)
		log.Info(byteutil.BytesToHexStr(rxbuf, rxlen))
		err = up.Ans8583UpCash(rxbuf, rxlen)
		if err == nil {
			log.Info("上送成功!")
		}
		up.Ea.PrintFields(up.Ea.Field_R)
		//log.Info(up.Ea.Field_R)
	}
	return err
}
func main() {

	fmt.Println("test...")

	//获取当前路径
	//file, _ := os.Getwd()
	// load config file, it's optional
	// or log.LoadConfiguration("./example.json", "json")
	// config file could be json or xml
	//mylog := log.NewDefaultLogger(log.INFO)
	//mylog.Info("hahaha  test ...")
	//mylog.LoadConfiguration("./logconfig1.json")
	log.LoadConfiguration("./logconfig.json")
	up8583.Test()
	//log.LOGGER("").Info("category Test info test ...")
	log.LOGGER("Test").Info("category Test info test ...")
	log.LOGGER("Test").Info("category Test info test message: %s", "new test msg")
	log.LOGGER("Test").Debug("category Test debug test ...")

	log.LOGGER("Test1").Debug("category Test1 debug test ...")

	// Other category not exist, test
	log.LOGGER("Other").Debug("category Other debug test ...")
	// socket log test
	//log.LOGGER("TestSocket").Debug("category TestSocket debug test ...")

	// original log4go test
	//// log level: FINE, DEBUG, TRACE, INFO, WARNING,ERROR, CRITICAL
	//log.AddFilter("file", log.DEBUG, log.NewFileLogWriter("test.log", true, true)) // INFO级别+输出到文件，并开启rotate

	// original log4go test
	log.Info("nomal info test ...")
	log.Debug("nomal debug test ...")
	configUP()

	//加载读取csv格式的文件\w\w
	log.Info("load data.csv file...")
	rfile, _ := os.Open("data.csv")
	reader := csv.NewReader(rfile)
	for {
		// Read返回的是一个数组，它已经帮我们分割了，
		record, err := reader.Read()
		// 如果读到文件的结尾，EOF的优先级居然比nil还高！
		if err == io.EOF {
			break
		} else if err != nil {
			fmt.Println("记录集错误:", err)
			break
		}
		records = append(records, record)
		for i := 0; i < len(record); i++ {
			//fmt.Print(record[i] + " ")
		}
		//fmt.Print("\n")
	}
	log.Info("==============load rec, number = %d", len(records))
	fmt.Println(records[0])

	fmt.Print("\n")
	name := ""
	fmt.Println("press any key to continue: ")
	fmt.Scanln(&name)
	//up.Frame8583QD()

	//recvstr := "007960000001386131003111080810003800010AC0001450021122130107200800085500323231333031343931333239303039393939393930363030313433303137303131393939390011000005190030004046F161A743497B32EAC760DF5EA57DF5900ECCE3977731A7EA402DDF0000000000000000CFF1592A"

	//recv := byteutil.HexStringToBytes(recvstr)
	//ret := up.Ea.Ans8583Fields(recv, len(recv))
	//if ret == 0 {
	// 	fmt.Println("解析成功")
	// 	up.Ea.PrintFields(up.Ea.Field_R)
	// } else {
	// 	fmt.Println("解析失败")
	// }

	err := QdProc()
	checkErr(err)
	lenth := len(records)
	carbin := ""
	field55 := ""
	date := ""
	money := 0
	succcount := 0
	if err == nil {

		for i := 0; i < lenth; i++ {
			log.Info("=============== count = %d", i+1)
			log.Info(records[i])
			carbin = records[i][6]
			t, _ := time.Parse("2006/01/02 15:04:05", records[i][5])
			//fmt.Println(t)
			date = t.Format("060102")
			field55 = records[i][8]
			float, err := strconv.ParseFloat(records[i][4], 32/64)
			checkErr(err)
			money = int(float * 100)
			err = UpCashProc(carbin, money, date, field55)
			checkErr(err)
			if err == nil {
				succcount++
				log.Info("=============== 上传成功,succ count = %d", succcount)
			}
			log.Info("=============== 记录流水= %d", up8583.RecSn)
			up8583.RecSn++
			//fmt.Println("press any key to next:")
			//fmt.Scanln(&name)
			//log.Info("cardbin,date,mondy,field=%s,%s,%d,%s", carbin, date, money, field55)
		}
		//err = QrcodeProc()
		//err = UpCashProc()
		//checkErr(err)

	}

	fmt.Println("over!press any key to continue: ")
	fmt.Scanln(&name)
	//UpCashProc()
	//getField55()

}
