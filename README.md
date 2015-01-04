Simple S3 API for Go. Docs [here](http://godoc.org/github.com/eaigner/s3).

#### S3 Config

The configuration contains you credentials and bucket info.

```
s3c := &s3.S3{
  Bucket:    os.Getenv("S3_BUCKET"),
  AccessKey: os.Getenv("S3_KEY"),
  Secret:    os.Getenv("S3_SECRET"),
  Path:      os.Getenv("S3_PATH"),
}
```

#### Object

`Object(path)` returns a new S3 object handle bound to the configuration it was created from.

```
obj := s3c.Object("path/to/hello.txt")
```

#### Upload

Writing to the `WriteAbortCloser` returned by `Writer()` allows you to upload objects.

```
w := obj.Writer()
io.Copy(w, bytes.NewBufferString("hello world!"))
w.Close()

// NOTE: You can abort uploads with w.Abort()
```

#### Download

Reading from the `ReadCloser` returned by `Reader()` allows you to download objects.

```
r, headers, err := obj.Reader()
b, err := ioutil.ReadAll(r)
```

#### Existence

Check if an object exists.

```
exists, err := obj.Exists()
```

#### Delete

Delete the object.

```
err := obj.Delete()
```

#### Generate Signed Form Upload URLs

```
o := s3c.Object("hello.txt")

p := make(s3.Policy)
p.SetExpiration(3600)
p.Conditions().Bucket(s3c.Bucket)
p.Conditions().ACL(s3.PublicRead)
p.Conditions().StartsWith("$key", "hello.txt")

url, err := o.FormURL(s3.PublicRead, p)
```

#### Generate Pre-Signed Expiring URLs

```
url, err := o.ExpiringURL(time.Second*60)
```
