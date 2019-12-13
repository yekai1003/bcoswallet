package cmd

import (
	"bcoswallet/erc20s"
	"bcoswallet/hd"
	"bcoswallet/hdkeystore"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/big"
	"strings"

	"log"
	"os"

	"github.com/howeyc/gopass"
	"github.com/tyler-smith/go-bip39"
	"github.com/yekai1003/gobcos/accounts/abi/bind"
	"github.com/yekai1003/gobcos/client"
	"github.com/yekai1003/gobcos/common"
)

type CLI struct {
	DataPath   string
	NetWorkUrl string
	TokenFile  string
}

type TokenConfig struct {
	Symbol string `json:"symbol"`
	Addr   string `json:"addr"`
}

func NewCLI(path, url, tokenfile string) *CLI {
	return &CLI{
		DataPath:   path,
		NetWorkUrl: url,
		TokenFile:  tokenfile,
	}
}

//提供帮助
func (c CLI) Help() {
	fmt.Println("./bcoswallet createwallet -name ACCOUNT_NAME --for create new wallet")
	fmt.Println("./bcoswallet balance -name ACCOUNT_NAME  --for get balance")
	fmt.Println("./bcoswallet sendtoken -name ACCOUNT_NAME -symbol SYMBOL -toaddr ADDRESS -amount AMOUNT --for get balance")
	fmt.Println("./bcoswallet addtoken -addr CONTRACT_ADDR --for add token to wallet")

}

//参数检测
func (c CLI) Valid() {
	if len(os.Args) < 2 {
		c.Help()
		os.Exit(-1)
	}
}

func (c CLI) Run() {
	//运行前先检测
	c.Valid()
	//(一) 创建账户
	//解析命令行
	//1. 分类设定
	createwalletCMD := flag.NewFlagSet("createwallet", flag.ExitOnError)
	//2. 设定要解析具体参数
	createwalletCMD_name := createwalletCMD.String("name", "yekai", "ACCOUNT_NAME")

	//(三) 查询token余额
	balanceCMD := flag.NewFlagSet("balance", flag.ExitOnError)
	balanceCMD_name := balanceCMD.String("name", "yekai", "ACCOUNT_NAME")
	//(四) 添加token
	addtokenCMD := flag.NewFlagSet("addtoken", flag.ExitOnError)
	addtokenCMD_addr := addtokenCMD.String("addr", "", "CONTRACT_ADDR")
	//(五) token转账
	sendtokenCMD := flag.NewFlagSet("sendtoken", flag.ExitOnError)
	sendtokenCMD_name := sendtokenCMD.String("name", "", "ACCOUNT_NAME")
	sendtokenCMD_symbol := sendtokenCMD.String("symbol", "", "SYMBOL")
	sendtokenCMD_toaddr := sendtokenCMD.String("toaddr", "", "ADDRESS")
	sendtokenCMD_amount := sendtokenCMD.Int64("amount", 0, "AMOUNT")

	switch os.Args[1] {
	case "createwallet":
		//真的解析参数
		err := createwalletCMD.Parse(os.Args[2:])
		if err != nil {
			fmt.Println("Failed to createwalletCMD.Parse", err)
			return
		}
	case "balance":
		err := balanceCMD.Parse(os.Args[2:])
		if err != nil {
			fmt.Println("Failed to balanceCMD.Parse", err)
			return
		}
	case "addtoken":
		err := addtokenCMD.Parse(os.Args[2:])
		if err != nil {
			fmt.Println("Failed to addtokenCMD.Parse", err)
			return
		}
	case "sendtoken":
		err := sendtokenCMD.Parse(os.Args[2:])
		if err != nil {
			fmt.Println("Failed to sendtokenCMD.Parse", err)
			return
		}
	default:
		c.Help()
		os.Exit(1)
	}

	//运行具体功能
	if createwalletCMD.Parsed() {
		//创建账户
		fmt.Println(*createwalletCMD_name)
		if *createwalletCMD_name == "" {
			fmt.Println("createwalletCMD_name can not null")
			return
		}
		//解决密码问题
		pass, _ := gopass.GetPasswd()
		c.CreateWallet(*createwalletCMD_name, string(pass))
	}

	//查询余额
	if balanceCMD.Parsed() {
		if *balanceCMD_name == "" {
			fmt.Println("balanceCMD_name can not null")
			return
		}
		//查询余额
		c.GetTokensBalance(*balanceCMD_name)
	}
	//添加token
	if addtokenCMD.Parsed() {
		if *addtokenCMD_addr == "" {
			fmt.Println("contract addr can not null")
			return
		}
		//代码实现
		c.AddToken(*addtokenCMD_addr)
	}
	// 发送token
	if sendtokenCMD.Parsed() {
		if *sendtokenCMD_symbol == "" || *sendtokenCMD_toaddr == "" || *sendtokenCMD_amount <= 0 {
			fmt.Println("sendtokenCMD params error")
			return
		}
		//代码实现
		c.SendToken(*sendtokenCMD_name, *sendtokenCMD_toaddr, *sendtokenCMD_symbol, *sendtokenCMD_amount)
	}
}

//创建账户功能 name = yekai  /data/yekai/0x------- /data/xiaohong/0x
func (c CLI) CreateWallet(name, pass string) {
	//1. NewEntropy 必须是32的整数倍，并且在128- 256之间

	entropy, _ := bip39.NewEntropy(128)
	//2. 助记词
	mnemonic, _ := bip39.NewMnemonic(entropy)
	fmt.Println(mnemonic)

	wallet, err := hd.NewFromMnemonic(mnemonic, "")
	if err != nil {
		log.Panic("Failed to NewFromMnemonic")
	}
	path, _ := hd.ParseDerivationPath("m/44'/60'/0'/0/0")
	account, err := wallet.Derive(path, true)
	if err != nil {
		log.Panic("Failed to Derive", err)
	}
	fmt.Println(account.Address.Hex())
	//PrivateKey(account accounts.Account) (*ecdsa.PrivateKey, error)
	privateKey, err := wallet.PrivateKey(account)
	if err != nil {
		log.Panic("Failed to PrivateKey", err)
	}
	//调用生成keystore对象

	hdks := hdkeystore.NewHDkeyStore(c.DataPath+name, privateKey)
	//StoreKey(filename string, key *Key, auth string)
	hdks.StoreKey(hdks.JoinPath(account.Address.Hex()), &hdks.Key, pass)

}

func (c CLI) getAddr(name string) string {
	infos, err := ioutil.ReadDir(c.DataPath + name)
	if err != nil {
		log.Panic("Failed to ReadDir", err)
	}
	for _, v := range infos {
		if !v.IsDir() {
			if strings.HasPrefix(v.Name(), "0x") {
				return v.Name()
			}
		}
	}
	return ""
}

func hex2bigInt(hex string) *big.Int {
	n := new(big.Int)
	n, _ = n.SetString(hex[2:], 16)
	return n
}

//添加token

func (c CLI) AddToken(addr string) {
	//0. 读取配置文件
	tokens := c.ReadToken()

	//0.1 校验没有重复的地址
	if c.CheckToken(addr, tokens) {
		fmt.Println("token is exists", addr)
		return
	}
	//1. 连接到网络
	client, err := client.Dial(c.NetWorkUrl, 1)
	if err != nil {
		log.Panic("Failed to client.Dial", err)
	}
	//2. 通过合约地址创建合约实例
	ins, err := erc20.NewErc20(common.HexToAddress(addr), client)
	if err != nil {
		log.Panic("Failed to erc20.NewErc20:", err)
	}
	opts := bind.CallOpts{
		From: common.HexToAddress(addr),
	}
	sym, err := ins.Symbol(&opts)
	if err != nil {
		log.Panic("Failed to ins.Symbol:", err)
	}
	//3. 写入配置文件
	tokens = append(tokens, TokenConfig{sym, addr})
	content, err := json.Marshal(tokens)
	if err != nil {
		log.Panic("Failed to json.Marshal:", err)
	}
	hdkeystore.WriteKeyFile(c.TokenFile, content)
}

func (c CLI) ReadToken() []TokenConfig {
	tokens := []TokenConfig{}
	data, err := ioutil.ReadFile(c.TokenFile)
	if err != nil {
		//log.Panic("Failed to ReadFile", err)
		fmt.Println("Failed to ReadFile")
	}
	if len(data) > 0 {
		err = json.Unmarshal(data, &tokens)
		if err != nil {
			log.Panic("Failed to Unmarshal", err)
		}
	}
	return tokens
}

func (c CLI) CheckToken(addr string, tokens []TokenConfig) bool {
	for _, token := range tokens {
		if addr == token.Addr {
			return true
		}
	}
	return false
}

func (c CLI) SendToken(acctname, toaddr, symbol string, amount int64) {
	//1. 连接到网络
	client, err := client.Dial(c.NetWorkUrl, 1)
	if err != nil {
		log.Panic("Failed to client.Dial", err)
	}
	defer client.Close()
	//2. 生成合约实例
	contract_addr := c.GetContractAddr(symbol)
	if contract_addr == "" {
		fmt.Println("symbol not exists", symbol)
		return
	}
	ins, err := erc20.NewErc20(common.HexToAddress(contract_addr), client)
	if err != nil {
		log.Panic("Failed to erc20.NewErc20", err)
	}
	//3. 设置签名
	hdks := hdkeystore.NewHDkeyStore(c.DataPath+acctname, nil)
	fromAddr := c.getAddr(acctname)
	_, err = hdks.GetKey(common.HexToAddress(fromAddr), hdks.JoinPath(fromAddr), "123")
	if err != nil {
		log.Panic("Failed to hdks.GetKey ", err)
	}
	opts := hdks.NewTransactOpts()
	//4. 合约调用
	//opts *bind.TransactOpts, to common.Address, value *big.Int
	value := big.NewInt(amount)
	_, err = ins.Transfer(opts, common.HexToAddress(toaddr), value)
	if err != nil {
		log.Panic("Failed to ins.Transfer ", err)
	}
}

func (c CLI) GetContractAddr(symbol string) string {
	data, err := ioutil.ReadFile(c.TokenFile)
	if err != nil {
		log.Panic("Failed ioutil.ReadFile ", err)
	}
	tokens := []TokenConfig{}
	err = json.Unmarshal(data, &tokens)
	if err != nil {
		log.Panic("Failed json.Unmarshal ", err)
	}

	for _, v := range tokens {
		if v.Symbol == symbol {
			return v.Addr
		}
	}

	return ""

}

func (cli *CLI) GetTokensBalance(acct_name string) {
	//先读取要获取哪些token
	data, err := ioutil.ReadFile(cli.TokenFile)
	if err != nil {
		log.Panic("failed to GetTokensBalance when ReadFile ", err)
	}
	tokens := []TokenConfig{}
	err = json.Unmarshal(data, &tokens)
	if err != nil {
		log.Panic("failed to GetTokensBalance when Unmarshal ", err)
	}
	//再读取账户下有哪些地址
	acctaddr := cli.getAddr(acct_name)

	opts := bind.CallOpts{
		From: common.HexToAddress(acctaddr),
	}

	//每种token，每个地址做一个处理
	for _, token := range tokens {
		amount, err := cli.getTokenBalance(token.Addr, acctaddr, &opts)
		if err != nil {
			fmt.Println("failed to getTokenBalance: ", err)
			continue
		}
		fmt.Printf("%s:%v\n", token.Symbol, amount)
	}
}

func (cli *CLI) getTokenBalance(contract, account string, opts *bind.CallOpts) (*big.Int, error) {
	client, err := client.Dial(cli.NetWorkUrl, 1)
	if err != nil {
		log.Panic("failed to getTokenBalance when dial", err)
	}
	ins, err := erc20.NewErc20(common.HexToAddress(contract), client)
	if err != nil {
		log.Panic("failed to getTokenBalance when NewErc20", err)
	}
	return ins.BalanceOf(opts, common.HexToAddress(account))
}
