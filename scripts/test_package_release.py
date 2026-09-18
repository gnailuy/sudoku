import json, pathlib, stat, tempfile, unittest
from scripts.package_release import package, verify
class PackageReleaseTest(unittest.TestCase):
    def setUp(self):
        self.temp=tempfile.TemporaryDirectory(); self.root=pathlib.Path(self.temp.name); self.binary=self.root/"built-sudoku"; self.binary.write_bytes(b"test executable\n"); self.binary.chmod(stat.S_IRUSR|stat.S_IWUSR|stat.S_IXUSR); self.output=self.root/"release"; self.sha="a"*40
    def tearDown(self): self.temp.cleanup()
    def create(self): package(self.binary,self.output,"gnailuy/sudoku","CI",42,self.sha)
    def test_round_trip(self):
        self.create(); verify(self.output,self.sha); manifest=json.loads((self.output/"manifest.json").read_text()); self.assertEqual(manifest["executable"],"sudoku"); self.assertEqual((self.output/"sudoku").stat().st_mode&0o777,0o755)
    def test_rejects_checksum_mismatch(self):
        self.create(); (self.output/"sudoku").write_bytes(b"changed")
        with self.assertRaisesRegex(ValueError,"checksum"): verify(self.output,self.sha)
    def test_rejects_wrong_identity_and_unsafe_path(self):
        self.create(); path=self.output/"manifest.json"; manifest=json.loads(path.read_text()); manifest["commit"]="b"*40; path.write_text(json.dumps(manifest))
        with self.assertRaisesRegex(ValueError,"expected commit"): verify(self.output,self.sha)
        manifest["commit"]=self.sha; manifest["executable"]="../sudoku"; path.write_text(json.dumps(manifest))
        with self.assertRaisesRegex(ValueError,"unsafe"): verify(self.output,self.sha)
    def test_rejects_missing_file(self):
        self.create(); (self.output/"sudoku").unlink()
        with self.assertRaisesRegex(ValueError,"missing"): verify(self.output,self.sha)
if __name__=="__main__": unittest.main()
