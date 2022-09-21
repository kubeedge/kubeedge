package backup

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

// Decode 解码消息
func Decode(reader *bufio.Reader) (string, error) {
	// 读取消息的长度
	lengthByte, _ := reader.Peek(2) // 读取前2个字节，看看包头
	lengthBuff := bytes.NewBuffer(lengthByte)
	var length int16
	// 读取实际的包体长度
	err := binary.Read(lengthBuff, binary.LittleEndian, &length)
	if err != nil {
		return "", err
	}
	// Buffered返回缓冲中现有的可读取的字节数。
	if int16(reader.Buffered()) < length+2 {
		return "", err
	}

	// 读取真正的消息数据
	realData := make([]byte, int(2+length))
	_, err = reader.Read(realData)
	if err != nil {
		return "", err
	}
	return string(realData[2:]), nil
}

func process(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	for {
		msg, err := Decode(reader)
		if err == io.EOF {
			return
		}
		if err != nil {
			fmt.Println("Decode error : ", err)
			return
		}
		fmt.Println("received data ：", msg)
	}
}

func server() {

	listen, err := net.Listen("tcp", "127.0.0.1:8888")
	if err != nil {
		fmt.Println("net.Listen error :", err)
		return
	}
	defer listen.Close()
	for {
		conn, err := listen.Accept()
		if err != nil {
			fmt.Println("listen.Accept error :", err)
			continue
		}
		go process(conn)
	}
}
