package contract

import (
    "google.golang.org/protobuf/proto"
    "google.golang.org/protobuf/reflect/protodesc"
    "google.golang.org/protobuf/reflect/protoregistry"
)

func init() {
    // File_tx_proto is already registered by tx.pb.go's own init()
    // Just serialize it and all its dependencies
    fd := File_tx_proto
    fdp := protodesc.ToFileDescriptorProto(fd)

    seen := map[string]bool{fd.Path(): true}
    result := [][]byte{}

    var addFile func(path string)
    addFile = func(path string) {
        f, err := protoregistry.GlobalFiles.FindFileByPath(path)
        if err != nil {
            return
        }
        fp := protodesc.ToFileDescriptorProto(f)
        b, err := proto.Marshal(fp)
        if err != nil {
            panic(err)
        }
        result = append(result, b)
        for _, dep := range fp.Dependency {
            if !seen[dep] {
                seen[dep] = true
                addFile(dep)
            }
        }
    }

    // Add deps first, then tx.proto itself
    for _, dep := range fdp.Dependency {
        if !seen[dep] {
            seen[dep] = true
            addFile(dep)
        }
    }
    b, _ := proto.Marshal(fdp)
    result = append(result, b)

    ContractConfig.FileDescriptorProtos = result
}
