package main

import (
	"fmt"
	"net"
)

var device FIDODevice

func handleCommandSubmit(conn *net.Conn, header USBIPMessageHeader, command USBIPCommandSubmitBody) {
	checkEOF(conn)
	transferBuffer := make([]byte, command.TransferBufferLength)
	if header.Direction == USBIP_DIR_OUT && command.TransferBufferLength > 0 {
		_, err := (*conn).Read(transferBuffer)
		checkErr(err, "Could not read transfer buffer")
	}
	switch command.Setup.BRequest {
	case USB_REQUEST_GET_DESCRIPTOR:
		checkEOF(conn)
		descriptor := device.getDescriptor(command.Setup.WValue)
		copy(transferBuffer, descriptor)
		replyHeader, replyBody, _ := newReturnSubmit(header, command, (transferBuffer))
		fmt.Printf("RETURN SUBMIT: %#v %#v %v %v %v\n\n", replyHeader, replyBody, toBE(replyHeader), toBE(replyBody), transferBuffer)
		write(*conn, toBE(replyHeader))
		write(*conn, toBE(replyBody))
		write(*conn, transferBuffer)
		checkEOF(conn)
	default:
		panic(fmt.Sprintf("Invalid CMD_SUBMIT bRequest: %d", command.Setup.BRequest))
	}
}

func handleCommands(conn *net.Conn) {
	for {
		checkEOF(conn)
		header := readBE[USBIPMessageHeader](*conn)
		fmt.Printf("MESSAGE HEADER: %#v\n\n", header)
		checkEOF(conn)
		if header.Command == USBIP_COMMAND_SUBMIT {
			command := readBE[USBIPCommandSubmitBody](*conn)
			fmt.Printf("COMMAND SUBMIT: %#v\n\n", command)
			handleCommandSubmit(conn, header, command)
		} else if header.Command == USBIP_COMMAND_UNLINK {
			unlink := readBE[USBIPCommandUnlinkBody](*conn)
			fmt.Printf("COMMAND UNLINK: %#v\n\n", unlink)
		} else {
			panic(fmt.Sprintf("Unsupported Command; %#v", header))
		}
	}
}

func handleConnection(conn *net.Conn) {
	for {
		header := readBE[USBIPControlHeader](*conn)
		fmt.Printf("USBIP CONTROL MESSAGE: %#v\n\n", header)
		checkEOF(conn)
		if header.CommandCode == USBIP_COMMAND_OP_REQ_DEVLIST {
			reply := newOpRepDevlist()
			fmt.Printf("OP_REP_DEVLIST: %#v\n\n", reply)
			write(*conn, toBE(reply))
		} else if header.CommandCode == USBIP_COMMAND_OP_REQ_IMPORT {
			busId := make([]byte, 32)
			bytesRead, err := (*conn).Read(busId)
			if bytesRead != 32 {
				panic(fmt.Sprintf("Could not read busId for OP_REQ_IMPORT: %v", err))
			}
			fmt.Println("BUS_ID: ", string(busId))
			reply := newOpRepImport()
			fmt.Printf("OP_REP_IMPORT: %#v\n\n", reply)
			write(*conn, toBE(reply))
			handleCommands(conn)
		}
	}
}

func main() {
	fmt.Println("Starting USBIP server...")
	device = FIDODevice{}
	listener, err := net.Listen("tcp", ":3240")
	if err != nil {
		fmt.Println("Could not create listener:", err)
		return
	}
	for {
		connection, err := listener.Accept()
		if err != nil {
			fmt.Println("Connection error:", err)
		}
		handleConnection(&connection)
	}
}