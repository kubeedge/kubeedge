package backup

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
)

// Encode 编码消息
func Encode(message string) ([]byte, error) {
	// 读取消息的长度，并且要 转换成int16类型（占2个字节） ，我们约定好的 包头2字节
	var length = int16(len(message))
	var nb = new(bytes.Buffer)

	// 写入消息头
	err := binary.Write(nb, binary.LittleEndian, length)
	if err != nil {
		return nil, err
	}

	// 写入消息体
	err = binary.Write(nb, binary.LittleEndian, []byte(message))
	if err != nil {
		return nil, err
	}
	return nb.Bytes(), nil
}

func client() {
	conn, err := net.Dial("tcp", "127.0.0.1:8888")
	if err != nil {
		fmt.Println("net.Dial error : ", err)
		return
	}
	defer conn.Close()
	for i := 0; i < 30; i++ {
		msg := `Hello world,hello xiaomotong!`

		data, err := Encode(msg)
		if err != nil {
			fmt.Println("Encode msg error : ", err)
			return
		}
		conn.Write(data)
	}
}
