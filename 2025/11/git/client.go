package git

import (
   "bytes"
   "fmt"
   "io"
   "net/http"
   "os"
   "strings"
)

// client represents the internal state for a git clone operation.
type client struct {
   repoURL    string
   directory  string
   refs       map[string]string
   headSHA    string
   headSymRef string
}

// Clone fetches a repository from a URL into a new directory.
func Clone(url, directory string) error {
   c := &client{
      repoURL:   url,
      directory: directory,
      refs:      make(map[string]string),
   }

   if err := os.MkdirAll(directory, 0755); err != nil {
      return fmt.Errorf("failed to create directory %s: %w", directory, err)
   }

   if err := c.discoverRefs(); err != nil {
      return fmt.Errorf("failed to discover refs: %w", err)
   }

   if err := c.fetchPackfile(); err != nil {
      return fmt.Errorf("failed to fetch packfile: %w", err)
   }

   if err := WriteRepo(c.directory, c.headSymRef, c.refs); err != nil {
      return fmt.Errorf("failed to write repository structure: %w", err)
   }

   fmt.Printf("Repository cloned into %s.\n", directory)
   return nil
}

// fetchPackfile handles the simple (non-multiplexed) packfile protocol.
func (c *client) fetchPackfile() error {
   var body bytes.Buffer
   wantLine := fmt.Sprintf("want %s\n", c.headSHA)
   fmt.Fprintf(&body, "%04x%s", len(wantLine)+4, wantLine)
   body.WriteString("0000")
   body.WriteString("0009done\n")

   uploadPackURL := fmt.Sprintf("%s/git-upload-pack", c.repoURL)
   req, err := http.NewRequest("POST", uploadPackURL, &body)
   if err != nil {
      return err
   }
   req.Header.Set("Content-Type", "application/x-git-upload-pack-request")

   res, err := http.DefaultClient.Do(req)
   if err != nil {
      return err
   }
   defer res.Body.Close()

   if res.StatusCode != http.StatusOK {
      return fmt.Errorf("unexpected status code from upload-pack: %d", res.StatusCode)
   }

   // Read and discard the initial ACK/NAK response line.
   _, err = ReadPktLine(res.Body)
   if err != nil {
      return fmt.Errorf("error reading initial ACK/NAK response: %w", err)
   }

   // Pass the rest of the stream to the parser.
   return ParsePackfile(c.directory, res.Body)
}

// discoverRefs performs the initial negotiation with the remote.
func (c *client) discoverRefs() error {
   res, err := http.Get(fmt.Sprintf("%s/info/refs?service=git-upload-pack", c.repoURL))
   if err != nil {
      return err
   }
   defer res.Body.Close()
   if res.StatusCode != http.StatusOK {
      return fmt.Errorf("unexpected status code from remote: %d", res.StatusCode)
   }
   body, err := io.ReadAll(res.Body)
   if err != nil {
      return err
   }
   reader := bytes.NewReader(body)
   _, err = ReadPktLine(reader)
   if err != nil {
      return err
   }
   var firstLine string
   for {
      pkt, err := ReadPktLine(reader)
      if err != nil {
         return err
      }
      if pkt.Length > 0 {
         firstLine = strings.TrimSpace(string(pkt.Payload))
         break
      }
   }
   parts := strings.Split(firstLine, "\x00")
   if len(parts) < 2 {
      return fmt.Errorf("invalid format for HEAD advertisement line: %q", firstLine)
   }
   headAndSha := strings.SplitN(parts[0], " ", 2)
   c.headSHA = headAndSha[0]
   capabilities := parts[1]
   for _, cap := range strings.Split(capabilities, " ") {
      if strings.HasPrefix(cap, "symref=HEAD:") {
         c.headSymRef = strings.TrimPrefix(cap, "symref=HEAD:")
         break
      }
   }
   c.refs[c.headSymRef] = c.headSHA
   for {
      pkt, err := ReadPktLine(reader)
      if err != nil {
         if err == io.EOF {
            break
         }
         return err
      }
      if pkt.Length == 0 {
         break
      }
      line := strings.TrimSpace(string(pkt.Payload))
      refParts := strings.SplitN(line, " ", 2)
      if len(refParts) == 2 {
         c.refs[refParts[1]] = refParts[0]
      }
   }
   if c.headSHA == "" || c.headSymRef == "" {
      return fmt.Errorf("could not determine HEAD commit hash or symbolic ref")
   }
   return nil
}
