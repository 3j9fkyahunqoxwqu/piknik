package main

import (
	"bufio"
	"crypto/subtle"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/minio/blake2b-simd"

	"golang.org/x/crypto/ed25519"
)

func getOperation(conf Conf, h1 []byte, reader *bufio.Reader, writer *bufio.Writer) {
	rbuf := make([]byte, 32)
	if _, err := io.ReadFull(reader, rbuf); err != nil {
		log.Print(err)
		return
	}
	h2 := rbuf
	hf2, _ := blake2b.New(&blake2b.Config{
		Key:    conf.Psk,
		Person: []byte(domainStr),
		Size:   32,
		Salt:   []byte{2},
	})
	hf2.Write(h1)
	wh2 := hf2.Sum(nil)
	if subtle.ConstantTimeCompare(wh2, h2) != 1 {
		return
	}

	storedContentRWMutex.RLock()
	signature := storedContent.signature
	ciphertextWithNonce := storedContent.ciphertextWithNonce
	storedContentRWMutex.RUnlock()

	hf3, _ := blake2b.New(&blake2b.Config{
		Key:    conf.Psk,
		Person: []byte(domainStr),
		Size:   32,
		Salt:   []byte{3},
	})
	hf3.Write(h2)
	hf3.Write(storedContent.signature)
	h3 := hf3.Sum(nil)
	writer.Write(h3)
	ciphertextWithNonceLen := uint64(len(ciphertextWithNonce))
	binary.Write(writer, binary.LittleEndian, ciphertextWithNonceLen)
	writer.Write(signature)
	writer.Write(ciphertextWithNonce)
	if err := writer.Flush(); err != nil {
		log.Print(err)
		return
	}
}

func storeOperation(conf Conf, h1 []byte, reader *bufio.Reader, writer *bufio.Writer) {
	rbuf := make([]byte, 104)
	if _, err := io.ReadFull(reader, rbuf); err != nil {
		log.Print(err)
		return
	}
	h2 := rbuf[0:32]
	ciphertextWithNonceLen := binary.LittleEndian.Uint64(rbuf[32:40])
	if conf.MaxLen > 0 && ciphertextWithNonceLen > conf.MaxLen {
		fmt.Printf("%v bytes requested to be stored, but limit set to %v bytes (%v Mb)\n",
			ciphertextWithNonceLen, conf.MaxLen, conf.MaxLen/(1024*1024))
		return
	}
	signature := rbuf[40:104]
	hf2, _ := blake2b.New(&blake2b.Config{
		Key:    conf.Psk,
		Person: []byte(domainStr),
		Size:   32,
		Salt:   []byte{2},
	})
	hf2.Write(h1)
	hf2.Write(signature)
	wh2 := hf2.Sum(nil)
	if subtle.ConstantTimeCompare(wh2, h2) != 1 {
		return
	}
	ciphertextWithNonce := make([]byte, ciphertextWithNonceLen)
	if _, err := io.ReadFull(reader, ciphertextWithNonce); err != nil {
		log.Print(err)
		return
	}
	if ed25519.Verify(conf.SignPk, ciphertextWithNonce, signature) != true {
		return
	}
	hf3, _ := blake2b.New(&blake2b.Config{
		Key:    conf.Psk,
		Person: []byte(domainStr),
		Size:   32,
		Salt:   []byte{3},
	})
	hf3.Write(h2)
	h3 := hf3.Sum(nil)

	storedContentRWMutex.Lock()
	storedContent.signature = signature
	storedContent.ciphertextWithNonce = ciphertextWithNonce
	storedContentRWMutex.Unlock()

	writer.Write(h3)
	if err := writer.Flush(); err != nil {
		log.Print(err)
		return
	}
}

func handleClient(conf Conf, conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	rbuf := make([]byte, 65)
	if _, err := io.ReadFull(reader, rbuf); err != nil {
		log.Print(err)
		return
	}
	version := rbuf[0]
	if version != 1 {
		return
	}
	r := rbuf[1:33]
	h0 := rbuf[33:65]
	hf0, _ := blake2b.New(&blake2b.Config{
		Key:    conf.Psk,
		Person: []byte(domainStr),
		Size:   32,
		Salt:   []byte{0},
	})
	hf0.Write([]byte{version})
	hf0.Write(r)
	wh0 := hf0.Sum(nil)
	if subtle.ConstantTimeCompare(wh0, h0) != 1 {
		return
	}
	hf1, _ := blake2b.New(&blake2b.Config{
		Key:    conf.Psk,
		Person: []byte(domainStr),
		Size:   32,
		Salt:   []byte{1},
	})
	hf1.Write([]byte{version})
	hf1.Write(h0)
	h1 := hf1.Sum(nil)
	writer := bufio.NewWriter(conn)
	writer.Write([]byte{version})
	writer.Write(h1)
	if err := writer.Flush(); err != nil {
		log.Print(err)
		return
	}
	operation, err := reader.ReadByte()
	if err != nil {
		return
	}
	switch operation {
	case byte('G'):
		getOperation(conf, h1, reader, writer)
	case byte('S'):
		storeOperation(conf, h1, reader, writer)
	}
}

// ServerMain - run a server
func ServerMain(conf Conf) {
	listen, err := net.Listen("tcp", conf.Listen)
	if err != nil {
		log.Fatal(err)
	}
	defer listen.Close()
	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go handleClient(conf, conn)
	}
}
