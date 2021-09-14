package rpcx

import (
	"fmt"

	"github.com/meshplus/bitxhub-model/pb"
)

func Int32(i int32) *pb.Arg {
	return generateArg(pb.Arg_I32, []byte(fmt.Sprintf("%d", i)))
}

func Int64(i int64) *pb.Arg {
	return generateArg(pb.Arg_I64, []byte(fmt.Sprintf("%d", i)))
}

func Uint32(i uint32) *pb.Arg {
	return generateArg(pb.Arg_U32, []byte(fmt.Sprintf("%d", i)))
}

func Uint64(i uint64) *pb.Arg {
	return generateArg(pb.Arg_U64, []byte(fmt.Sprintf("%d", i)))
}

func Float32(f float32) *pb.Arg {
	return generateArg(pb.Arg_F32, []byte(fmt.Sprintf("%g", f)))
}

func Float64(f float64) *pb.Arg {
	return generateArg(pb.Arg_F64, []byte(fmt.Sprintf("%g", f)))
}

func String(content string) *pb.Arg {
	return generateArg(pb.Arg_String, []byte(content))
}

func Bytes(content []byte) *pb.Arg {
	return generateArg(pb.Arg_Bytes, content)
}

func Bool(b bool) *pb.Arg {
	return generateArg(pb.Arg_Bool, []byte(fmt.Sprintf("%v", b)))
}

func generateArg(typ pb.Arg_Type, content []byte) *pb.Arg {
	return &pb.Arg{
		Type:  typ,
		Value: content,
	}
}
