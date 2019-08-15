package main

import (
	"flag"
	"fmt"
	"go8583/byteutil"
	"go8583/netutil"
	"go8583/up8583"
	"io"
	"os"
	"strconv"

	log "github.com/jeanphorn/log4go"
	"github.com/larspensjo/config"
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
	records []string
	Url     string = ""
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
	Url, err = cfg.String("SERVERCONFIG", "Url")
	log.Info("Server:%s\n", Server)
	log.Info("Port:%d\n", Port)
	log.Info("Urk:%s\n", Url)
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
	up.Frame8583QD()
	up.Ea.PrintFields(up.Ea.Field_S)

	if Url == "" {
		log.Info("connect:server=%s,port=%d\n", Server, Port)
		conn, err := netutil.Connect(Server, Port)
		if err == nil {
			log.Info("connect ok!")
		} else {
			log.Info("connect failed!")
			return err
		}
		defer netutil.DisConnect(conn)
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
	} else {
		rxbuf, err := netutil.UpHttpsPost(Url, up.Ea.Txbuf)
		rxlen := len(rxbuf)
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
/*
0000AAAB//记录流水 4字节
07 //记录类型 1字节（银联电子现金记录，类型为07）
000443 //批次号 3字节
00
13//发卡行应用数据长度
13//主账号长度
62159837600086503580 //10字节
3030303030303533  //终端号8字节
383938353030303832313130313036 //商户号
110731//交易时间 3字节
86C97AE822DEC282//应用密文8字节
40//应用信息数据 1字节
07010103900000010A010000000000B0
81E4D86215983760008650358D260722 //发卡行应用数据 32字节
DD29173D//随机数 4字节
01FB //应用交易计数器
0000000000//终端验证结果5字节
180919//交易日期 3字节
00//交易类型1字节
000000000200//交易金额6字节
0156//交易货币代码 2字节
7C00//应用交互特征 2字节
0156//终端国家代码 2字节
000000000000 //其他金额 6字节
04//交易序列计数器长度 1字节
00043691//交易序列计数器 4字节
01260731 //卡有效期 4字节
01//持卡序号长度
01//持卡序号(应用主账号序列号)
3030303030303030 //接口设备序列号8字节
08//专用文件名称长度
A0000003330101010000000000000000//专用文件名称
008C//软件版本号
000000//持卡人验证方法
36//终端类型
00//卡标识信息长度
00000000000000000000000000000000//卡产品标识信息
06//电子现金发卡行授权码长度
454343303031//电子现金发卡行授权码(6字节) (到此处结束，解析完毕)
000000000000000000000000000000000000000000
00C800C80000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000
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
	//st := "0000AAAB0700044300131362159837600086503580303030303030353338393835303030383231313031303611073186C97AE822DEC2824007010103900000010A010000000000B081E4D86215983760008650358D260722DD29173D01FB00000000001809190000000000020001567C0001560000000000000400043691012607310101303030303030303008A0000003330101010000000000000000008C0000003600000000000000000000000000000000000645434330303100000000000000000000000000000000000000000000C800C80000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000A28E"
	//st := "00000000483f7ee5314c8565c300000000019000000000000000000000000156015600611d181c7c00009a13071e0103900000010a010000010145a1f302076013827504000669696d27042240e0e1c80f270400021030313034353530300000000000006000064543433030310a60138275040000"
	st := filed55
	//money := "190" //1.9元
	//cardbin := "601382750400038"
	//l := 0
	s := ""

	//l += 5 * 2

	s += "9F2608"
	s += st[47*2 : 47*2+8*2]

	fmt.Println(s)

	s += "9F0206"
	s += st[103*2 : 103*2+6*2]

	s += "9F0306"
	s += st[115*2 : 115*2+6*2]

	s += "9505"
	s += st[94*2 : 94*2+5*2]

	s += "9F1A02"
	s += st[113*2 : 113*2+2*2]

	s += "5F2A02"
	s += st[109*2 : 109*2+2*2]

	s += "9C01"
	s += st[102*2 : 102*2+1*2]

	s += "9F3704"
	s += st[88*2 : 88*2+4*2]

	s += "8202"
	s += st[111*2 : 111*2+2*2]

	s += "9F3602"
	s += st[92*2 : 92*2+2*2]

	s += "9F10"
	tmplen, _ := strconv.ParseInt(st[9*2:9*2+1*2], 16, 16)
	s += st[9*2 : 9*2+1*2]

	s += st[56*2 : 56*2+int(tmplen)*2]

	s += "9F2701"
	s += st[55*2 : 55*2+1*2]

	s += "9F3303"
	s += "000000"

	//卡有效期
	cardvailddata = st[127*2 : 127*2+2*2]

	fmt.Printf("cardvailddata:%s\n", cardvailddata)
	//卡片序列号
	cardholdsn = ""
	if st[130*2:130*2+1*2] != "00" {
		cardholdsn = "00"
		cardholdsn += st[131*2 : 131*2+1*2]
	}

	fmt.Printf("cardholdsn:%s\n", cardholdsn)
	//卡产品标识
	if st[163*2:163*2+1*2] != "00" {
		s += "9F63"
		tmplen, _ = strconv.ParseInt(st[163*2:163*2+1*2], 16, 16)
		s += st[163*2 : 163*2+1*2]
		s += st[164*2 : 164*2+int(tmplen)*2]
	}

	//电子现金发卡行授权码9F74
	s += "9F74"
	tmplen, _ = strconv.ParseInt(st[180*2:180*2+1*2], 16, 16)
	s += st[180*2 : 180*2+1*2]
	s += st[181*2 : 181*2+int(tmplen)*2]

	//fmt.Println(st)
	s += "9A03"
	s += st[99*2 : 99*2+3*2]
	fmt.Printf("date=%s\n", st[99*2:99*2+3*2])
	//s += "181205"
	//s += date
	s += "9F1E083030303030303030"
	s += "9f0902008c"
	s += "9f3403000000"
	s += "9f350136"
	s += "8A025931"
	s += "9F4104"
	s += fmt.Sprintf("%08d", up8583.RecSn)
	//fmt.Println(s)

	return byteutil.HexStringToBytes(s)
}

func UpCashProc(carbin string, money int, rec string) error {

	field55 := getField55(rec, "")
	fmt.Printf("carbin=%s\n", carbin)
	//cardbin = "601382750400038"
	//cardbin = "601382750400074"
	up.Frame8583UpCash(carbin, money, cardvailddata, cardholdsn, field55)
	up.Ea.PrintFields(up.Ea.Field_S)
	log.Info(byteutil.BytesToHexString(up.Ea.Txbuf))
	if Url == "" {
		fmt.Printf("connect:server=%s,port=%d\n", Server, Port)
		conn, err := netutil.Connect(Server, Port)
		if err == nil {
			fmt.Println("connect ok!")
		} else {
			fmt.Println("connect failed!")
			return err
		}
		defer netutil.DisConnect(conn)

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
	} else {
		rxbuf, err := netutil.UpHttpsPost(Url, up.Ea.Txbuf)
		rxlen := len(rxbuf)
		if err == nil {
			log.Info("recv ok!len=%d\n", rxlen)
			log.Info(byteutil.BytesToHexStr(rxbuf, rxlen))
			err = up.Ans8583UpCash(rxbuf, rxlen)
			if err == nil {
				log.Info("上送成功!")
			}
			up.Ea.PrintFields(up.Ea.Field_R)
		}
		return err
	}

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
	log.Info("load RecFile.txt file...")
	rfile, _ := os.Open("RecFile_0.txt")
	buf := make([]byte, 600)
	defer rfile.Close()
	for {
		// Read返回的是一个数组，它已经帮我们分割了，
		n, err := rfile.Read(buf)
		// 如果读到文件的结尾，EOF的优先级居然比nil还高！
		if err == io.EOF {
			break
		} else if err != nil {
			fmt.Println("记录集错误:", err)
			break
		}
		if n != 600 {
			fmt.Println("记录长度错误!")
			//fmt.Println(string(record))
			break
		}
		record := string(buf)
		fmt.Println(record)
		fmt.Println(record[8:10])
		if record[8:10] == "17" {
			continue
		}
		if record[8:10] == "27" {
			continue
		}
		if record[8:10] == "FF" {
			fmt.Println("bad record!!")
			break
		}
		if record[8:10] == "07" {
			records = append(records, record)
			fmt.Printf("=============================date=%s\n", record[99*2:99*2+3*2])
		}
		//break
		//records = append(records, record)

	}
	log.Info("==============load rec, number = %d", len(records))
	fmt.Println(records[0])

	fmt.Print("\n")
	//st := "0000AAAB0700044300131362159837600086503580303030303030353338393835303030383231313031303611073186C97AE822DEC2824007010103900000010A010000000000B081E4D86215983760008650358D260722DD29173D01FB00000000001809190000000000020001567C0001560000000000000400043691012607310101303030303030303008A0000003330101010000000000000000008C0000003600000000000000000000000000000000000645434330303100000000000000000000000000000000000000000000C800C80000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000A28E"
	//date := ""
	//getField55(st, date)
	name := ""
	fmt.Println("press any key to continue: ")
	fmt.Scanln(&name)
	err := QdProc()
	checkErr(err)
	lenth := len(records)
	cardbin := ""
	//field55 := ""
	//date := ""
	money := 0
	succcount := 0
	if err == nil {

		for i := 0; i < lenth; i++ {
			log.Info("=============== count = %d", i+1)
			log.Info(records[i])
			//tmprec := records[i]
			tmplen, _ := strconv.ParseInt(records[i][10*2:10*2+1*2], 16, 16)
			log.Info("tmplen:%d", tmplen)

			cardbin = records[i][11*2 : 11*2+tmplen]
			tmpmoney := records[i][103*2 : 103*2+6*2]
			log.Info("cardbin:%s", cardbin)
			log.Info("money:%s", tmpmoney)
			float, err := strconv.ParseFloat(tmpmoney, 32/64)
			checkErr(err)
			money = int(float)
			log.Info("money1:%d", money)
			err = UpCashProc(cardbin, money, records[i])
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
