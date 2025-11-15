package git

import (
   "bufio"
   "bytes"
   "compress/zlib"
   "encoding/binary"
   "encoding/hex"
   "fmt"
   "io"
)

// Constants for Git object types found in packfiles.
const (
   _           = iota // 0 is not a valid object type
   objCommit          // 1
   objTree            // 2
   objBlob            // 3
   objTag             // 4
   _                  // 5 is unused
   objOfsDelta        // 6 (Offset Delta) - Not implemented
   objRefDelta        // 7 (Reference Delta)
)

// ParsePackfile reads a packfile stream, decodes the objects, and writes them.
func ParsePackfile(directory string, r io.Reader) error {
   bufReader := bufio.NewReader(r)

   var header [12]byte
   if _, err := io.ReadFull(bufReader, header[:]); err != nil {
      return fmt.Errorf("failed to read packfile header: %w", err)
   }
   if string(header[:4]) != "PACK" || binary.BigEndian.Uint32(header[4:8]) != 2 {
      return fmt.Errorf("invalid packfile header or version")
   }
   numObjects := binary.BigEndian.Uint32(header[8:12])

   for i := uint32(0); i < numObjects; i++ {
      b, err := bufReader.ReadByte()
      if err != nil {
         return fmt.Errorf("failed to read object header byte: %w", err)
      }

      objType := (b >> 4) & 0x7
      size := int64(b & 0xf)
      var shift uint = 4
      for b&0x80 != 0 {
         b, err = bufReader.ReadByte()
         if err != nil {
            return fmt.Errorf("failed to read object size continuation byte: %w", err)
         }
         size |= int64(b&0x7f) << shift
         shift += 7
      }

      switch objType {
      case objCommit, objTree, objBlob, objTag:
         typeString := objectTypeToString(objType)
         objData, err := readZlibData(bufReader)
         if err != nil {
            return fmt.Errorf("failed to read object data for type %s: %w", typeString, err)
         }
         if _, err := WriteLooseObject(directory, typeString, objData); err != nil {
            return fmt.Errorf("failed to write loose object for type %s: %w", typeString, err)
         }
      case objRefDelta:
         var baseSha [20]byte
         if _, err := io.ReadFull(bufReader, baseSha[:]); err != nil {
            return fmt.Errorf("failed to read base SHA for REF_DELTA: %w", err)
         }
         deltaData, err := readZlibData(bufReader)
         if err != nil {
            return fmt.Errorf("failed to decompress delta data: %w", err)
         }

         newObjectData, baseType, err := applyDelta(directory, hex.EncodeToString(baseSha[:]), deltaData)
         if err != nil {
            return fmt.Errorf("failed to apply delta: %w", err)
         }
         if _, err := WriteLooseObject(directory, baseType, newObjectData); err != nil {
            return fmt.Errorf("failed to write reconstructed delta object: %w", err)
         }
      default:
         return fmt.Errorf("unsupported object type in packfile: %d", objType)
      }
   }
   return nil
}

// applyDelta reads a base object, applies a patch, and returns the new object data.
func applyDelta(directory, baseShaHex string, deltaData []byte) ([]byte, string, error) {
   baseObjectData, baseType, err := ReadLooseObject(directory, baseShaHex)
   if err != nil {
      return nil, "", fmt.Errorf("could not read base object %s: %w", baseShaHex, err)
   }

   deltaReader := bytes.NewReader(deltaData)
   _, err = readVarint(deltaReader) // Read and discard base size
   if err != nil {
      return nil, "", err
   }
   newSize, err := readVarint(deltaReader) // Read new object size
   if err != nil {
      return nil, "", err
   }

   var result bytes.Buffer
   for deltaReader.Len() > 0 {
      cmd, _ := deltaReader.ReadByte()
      if cmd&0x80 != 0 { // Copy from base object
         var offset, size uint32
         // This loop replaces the repetitive if statements
         for i := uint(0); i < 4; i++ {
            if cmd&(1<<i) != 0 {
               b, _ := deltaReader.ReadByte()
               offset |= uint32(b) << (i * 8)
            }
         }
         for i := uint(0); i < 3; i++ {
            if cmd&(1<<(i+4)) != 0 {
               b, _ := deltaReader.ReadByte()
               size |= uint32(b) << (i * 8)
            }
         }
         if size == 0 {
            size = 0x10000
         }
         result.Write(baseObjectData[offset : offset+size])
      } else { // Add new data
         size := cmd & 0x7f
         io.CopyN(&result, deltaReader, int64(size))
      }
   }

   if int64(result.Len()) != newSize {
      return nil, "", fmt.Errorf("delta result size mismatch: expected %d, got %d", newSize, result.Len())
   }

   return result.Bytes(), baseType, nil
}

// Helper functions
func readZlibData(r *bufio.Reader) ([]byte, error) {
   zr, err := zlib.NewReader(r)
   if err != nil {
      return nil, err
   }
   defer zr.Close()
   return io.ReadAll(zr)
}

func readVarint(r *bytes.Reader) (int64, error) {
   var result int64
   var shift uint
   for {
      b, err := r.ReadByte()
      if err != nil {
         return 0, err
      }
      result |= int64(b&0x7f) << shift
      shift += 7
      if b&0x80 == 0 {
         break
      }
   }
   return result, nil
}

func objectTypeToString(objType byte) string {
   switch objType {
   case objCommit:
      return "commit"
   case objTree:
      return "tree"
   case objBlob:
      return "blob"
   case objTag:
      return "tag"
   default:
      return "unknown"
   }
}
