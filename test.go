/**
 * Any code, applications, scripts, templates, proofs of concept,
 * documentation and other items are provided for illustration purposes only.
 *
 * (C) Copyright 2017 Amazon Web Services
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at:
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"bufio"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	flowlogs "github.com/vpc-flow-logs-capn-proto/vpc"

	"zombiezen.com/go/capnproto2"
)

func main() {

	writeCapnpFile()

	readCapnpFile()
}

func readCapnpFile() {

	file, err := os.Open("capnp.gz")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	gzIn, err := gzip.NewReader(file)
	if err != nil {
		panic(err)
	}
	defer gzIn.Close()

	decoder := capnp.NewPackedDecoder(gzIn)

	counter := 0

	for {

		msg, err := decoder.Decode()

		if err != nil {
			if !strings.Contains(err.Error(), "EOF") {
				panic(err)
			}
			break
		}

		_, err = flowlogs.ReadRootVpcFlowLogEntry(msg)
		if err != nil {
			panic(err)
		}

		counter++
	}

	fmt.Println(counter)
}

func writeCapnpFile() {

	file, err := os.Open("raw.txt.gz")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	out, err := os.OpenFile("capnp.gz", os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0660)
	if err != nil {
		panic(err)
	}
	defer out.Close()

	gzOut, err := gzip.NewWriterLevel(out, gzip.BestCompression)
	if err != nil {
		panic(err)
	}
	defer gzOut.Close()
	gzOutBuffer := bufio.NewWriter(gzOut)

	gzIn, err := gzip.NewReader(file)
	if err != nil {
		panic(err)
	}
	defer gzIn.Close()

	packedEncoder := capnp.NewPackedEncoder(gzOutBuffer)

	counter := 0

	scanner := bufio.NewScanner(gzIn)
	for scanner.Scan() {
		counter++

		line := scanner.Text()
		//fmt.Println(line)

		values := strings.Split(line, " ")

		msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
		if err != nil {
			panic(err)
		}

		entry, err := flowlogs.NewRootVpcFlowLogEntry(seg)

		if err != nil {
			panic(err)
		}

		version, err := strconv.ParseInt(values[0], 10, 8)
		if err != nil {
			panic(err)
		}
		entry.SetVersion(int8(version))

		accountId, err := strconv.ParseUint(values[1], 10, 64)
		if err != nil {
			panic(err)
		}
		entry.SetAccountId(accountId)

		entry.SetInterfaceId(values[2])

		if !strings.Contains(line, "NODATA") && !strings.Contains(line, "SKIPDATA") {

			entry.SetSrcAddr(ipStringToInt(values[3]))
			entry.SetDstAddr(ipStringToInt(values[4]))

			srcPort, err := strconv.ParseUint(values[5], 10, 16)
			if err != nil {
				panic(err)
			}
			entry.SetSrcPort(uint16(srcPort))

			dstPort, err := strconv.ParseUint(values[6], 10, 16)
			if err != nil {
				panic(err)
			}
			entry.SetDstPort(uint16(dstPort))

			protocol, err := strconv.ParseUint(values[7], 10, 8)
			if err != nil {
				panic(err)
			}
			entry.SetProtocol(uint8(protocol))

			packets, err := strconv.ParseUint(values[8], 10, 16)
			if err != nil {
				panic(err)
			}
			entry.SetPackets(uint16(packets))

			bytes, err := strconv.ParseUint(values[9], 10, 64)
			if err != nil {
				panic(err)
			}
			entry.SetBytes(bytes)
		}

		start, err := strconv.ParseUint(values[10], 10, 32)
		if err != nil {
			panic(err)
		}
		entry.SetStart(uint32(start))

		end, err := strconv.ParseUint(values[11], 10, 32)
		if err != nil {
			panic(err)
		}
		entry.SetEnd(uint32(end))

		if values[12] == "ACCEPT" {
			entry.SetAction(flowlogs.VpcFlowLogEntry_Action_accept)
		} else {
			entry.SetAction(flowlogs.VpcFlowLogEntry_Action_reject)
		}

		if values[13] == "OK" {
			entry.SetLogStatus(flowlogs.VpcFlowLogEntry_LogStatus_ok)
		} else if values[13] == "NODATA" {
			entry.SetLogStatus(flowlogs.VpcFlowLogEntry_LogStatus_noData)
		} else {
			entry.SetLogStatus(flowlogs.VpcFlowLogEntry_LogStatus_skipData)
		}

		err = packedEncoder.Encode(msg)
		if err != nil {
			panic(err)
		}
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}

	fmt.Println(counter)

	gzOutBuffer.Flush()

	gzOut.Flush()

}

func ipStringToInt(str string) uint32 {

	ip := net.ParseIP(str)

	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16])
	}
	return binary.BigEndian.Uint32(ip)
}
