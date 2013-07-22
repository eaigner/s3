Simple S3 API for Go. Docs [here](http://godoc.org/github.com/eaigner/s3).

#### Acknowledgements

First, a huge thanks goes to [Keith Rarick](https://github.com/kr) and [Lye](https://github.com/lye) on whose s3 packages this implementation is based on for a large part. The goal of this package is simply to provide a more convenient API.

#### S3 Config

The configuration contains you credentials and bucket info.

```
s3c := &s3.S3{
  Bucket: os.Getenv("S3_BUCKET"),
  Key:    os.Getenv("S3_KEY"),
  Secret: os.Getenv("S3_SECRET"),
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
w, err := obj.Writer()
defer w.Close()
io.Copy(w, bytes.NewBufferString("hello world!"))

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

#### Generate Signed URLs

```
o := s3c.Object("hello.txt")

p := make(s3.Policy)
p.SetExpiration(3600)
p.Conditions().AddBucket(s3c.Bucket)
p.Conditions().AddACL(s3.PublicRead)
p.Conditions().MatchStartsWith("$key", "hello.txt")

url, err := o.FormUploadURL(s3.PublicRead, p)
```
