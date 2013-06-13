#### Acknowledgements

First, a huge thanks goes to [Keith Rarick](https://github.com/kr) and [Lye](https://github.com/lye) on whose s3 packages this implementation is based on for a large part. The goal of this package is simply to provide a more convenient API.

#### 1. Create Config

The configuration contains you credentials and bucket info.

```
conf := &s3.Conf{
  Bucket: os.Getenv("S3_BUCKET"),
  Key:    os.Getenv("S3_KEY"),
  Secret: os.Getenv("S3_SECRET"),
}
```

#### 2. Get Object Handle

`Object(path)` returns a new S3 object bound to the configuration it was created from.

```
obj := conf.Object("path/to/hello.txt")
```

#### 3. Upload

Writing to the `WriteCloser` returned by `Writer()` allows you to upload objects.

```
w, err := obj.Writer()
defer w.Close()
io.Copy(w, bytes.NewBufferString("hello world!"))
```

#### 4. Download

Reading from the `ReadCloser` returned by `Reader()` allows you to download objects.

```
r, headers, err := obj.Reader()
b, err := ioutil.ReadAll(r)
```

#### 5. Delete

Delete the object.

```
err := obj.Delete()
```