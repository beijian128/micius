package crypto_test

import (
	"fmt"
	"github/beijian128/micius/frame/network/crypto"
	"io"
	"net"
	"sync"
	"testing"
)

func TestCrypto(t *testing.T) {
	c := crypto.NewAesCryptoUseDefaultKey()
	text := []byte("hello world")
	c.Encrypt(text, text)

	if string(text) == "hello world" {
		t.Fatal()
	}

	c.Decrypt(text, text)

	if string(text) != "hello world" {
		t.Fatal()
	}
}

func TestCrypto2(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			c := crypto.NewAesCryptoUseDefaultKey()
			text := []byte("hello world" + fmt.Sprint(i))

			for ii := 0; ii < 100; ii++ {
				c.Encrypt(text, text)

				if string(text) == fmt.Sprint("hello world", i) {
					t.Fail()
				}

				c.Decrypt(text, text)

				if string(text) != fmt.Sprint("hello world", i) {
					t.Fail()
				}
			}
		}()
	}
	wg.Wait()
}

func startTestServer(t *testing.T, addrChan chan string) {
	aesTool := crypto.NewAesCryptoUseDefaultKey()
	listener, err := net.Listen("tcp", "localhost:0") // 自动选择端口
	if err != nil {
		t.Error("Error listening:", err)
		return
	}
	defer listener.Close()

	// 发送实际监听的地址和端口到通道
	addrChan <- listener.Addr().String()

	conn, err := listener.Accept()
	if err != nil {
		t.Error("Error accepting:", err)
		return
	}
	defer conn.Close()

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		if err != io.EOF {
			t.Error("Read error:", err)
			return
		}
		return
	}

	// 解密消息
	decrypted := make([]byte, n)
	err = aesTool.Decrypt(buf[:n], decrypted)
	if err != nil {
		t.Error("Decrypt error:", err)
		return
	}
	t.Log(string(decrypted))
	// 加密并发送相同的消息回客户端
	encrypted := make([]byte, len(decrypted))
	aesTool.Encrypt(decrypted, encrypted)

	conn.Write(encrypted)
}

// 客户端逻辑，发送加密消息并验证响应
func testClient(t *testing.T, address string) {
	aesTool := crypto.NewAesCryptoUseDefaultKey()
	conn, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatal("Error connecting:", err)
	}
	defer conn.Close()

	message := "hello world"
	encryptedMsg := make([]byte, len([]byte(message)))
	aesTool.Encrypt([]byte(message), encryptedMsg)

	// 发送加密消息
	conn.Write(encryptedMsg)

	// 读取并解密响应
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		t.Error("Read error:", err)
		return
	}

	decryptedMsg := make([]byte, n)
	err = aesTool.Decrypt(buf[:n], decryptedMsg)
	if err != nil {
		t.Error("Decrypt error:", err)
		return
	}

	if string(decryptedMsg) != message {
		t.Errorf("Expected %s, got %s", message, string(decryptedMsg))
		return
	}
}

// 单元测试入口
func TestTcpCommunication(t *testing.T) {
	addrChan := make(chan string)
	go startTestServer(t, addrChan)

	// 从通道获取服务器地址
	addr := <-addrChan

	// 无需等待，直接使用服务器的实际地址进行连接

	testClient(t, addr)
}
