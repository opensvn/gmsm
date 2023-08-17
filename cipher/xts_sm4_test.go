package cipher_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/emmansun/gmsm/cipher"
	"github.com/emmansun/gmsm/sm4"
)

var xtsTestVectors = []struct {
	key        string
	sector     uint64
	plaintext  string
	ciphertext string
}{
	{ // XTS-SM4-128 applied for a data unit of 32 bytes
		"0000000000000000000000000000000000000000000000000000000000000000",
		0,
		"0000000000000000000000000000000000000000000000000000000000000000",
		"d9b421f731c894fdc35b77291fe4e3b02a1fb76698d59f0e51376c4ada5bc75d",
	}, {
		"1111111111111111111111111111111122222222222222222222222222222222",
		0x3333333333,
		"4444444444444444444444444444444444444444444444444444444444444444",
		"a74d726c11196a32be04e001ff29d0c7932f9f3ec29bfcb64dd17f63cbd3ea31",
	}, {
		"fffefdfcfbfaf9f8f7f6f5f4f3f2f1f022222222222222222222222222222222",
		0x3333333333,
		"4444444444444444444444444444444444444444444444444444444444444444",
		"7f76088effadf70c02ea9f95da0628d351bfcb9eac0563bcf17b710dab0a9826",
	}, { // XTS-SM4-128 applied for a data unit of 512 bytes
		"2718281828459045235360287471352631415926535897932384626433832795",
		0,
		"000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f404142434445464748494a4b4c4d4e4f505152535455565758595a5b5c5d5e5f606162636465666768696a6b6c6d6e6f707172737475767778797a7b7c7d7e7f808182838485868788898a8b8c8d8e8f909192939495969798999a9b9c9d9e9fa0a1a2a3a4a5a6a7a8a9aaabacadaeafb0b1b2b3b4b5b6b7b8b9babbbcbdbebfc0c1c2c3c4c5c6c7c8c9cacbcccdcecfd0d1d2d3d4d5d6d7d8d9dadbdcdddedfe0e1e2e3e4e5e6e7e8e9eaebecedeeeff0f1f2f3f4f5f6f7f8f9fafbfcfdfeff000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f404142434445464748494a4b4c4d4e4f505152535455565758595a5b5c5d5e5f606162636465666768696a6b6c6d6e6f707172737475767778797a7b7c7d7e7f808182838485868788898a8b8c8d8e8f909192939495969798999a9b9c9d9e9fa0a1a2a3a4a5a6a7a8a9aaabacadaeafb0b1b2b3b4b5b6b7b8b9babbbcbdbebfc0c1c2c3c4c5c6c7c8c9cacbcccdcecfd0d1d2d3d4d5d6d7d8d9dadbdcdddedfe0e1e2e3e4e5e6e7e8e9eaebecedeeeff0f1f2f3f4f5f6f7f8f9fafbfcfdfeff",
		"54dd65b6326faea8fad1a83c63614af39f721d8dfe177a30b66abf6a449980e1cdbe06afb73336f37a4d39de964a30d7d04a3799169c60258f6b748a61861aa5ec92a2c15b2b7c615a42aba499bbd6b71db9c789b2182089a25dd3df800ed1864d19f7ed45fd17a9480b0fb82d9b7fc3ed57e9a1140eaa778dd2dd679e3edc3dc4d55c950ebc531d9592f7c4638256d56518292a20af98fdd3a63600350a70ab5a40f4c285037ca01f251f19ecae0329ff77ad88cd5a4cdea2aeabc22148ffbd239bd10515bde1131dec8404e443dc763140d5f22bf33e0c6872d6b81d630f6f00cdd058fe80f9cbfb77707f93cee2ca92b915b8304027c190a84e2d65e018cc6a387d3766acdb28253284e8db9acf8f52280ddc6d0033d2ccaaa4f9aeff123669bc024fd6768edf8bc1f8d622c19c609ef97f609190cd110241e7fb084ed8942da1f9b9cf1b514b61a388b30ea61a4a745b381ee7ad6c4db1275453b8413f98df6e4a40986ee4b59af5dfaecd301265179067a00d7ca35ab95abd617adea28ec1c26a97de28b8bfe30120d6aefbd258c59e42d161e8065a78106bdca5cd90fb3aac4e93866c8a7f9676860a79145bd92e02e819a90be0b97cc522b32106856fdf0e54d88e4624155a2f1c14eaeaa163f858e99a806e791acd82f1b0e29f0028a4c38e976f571a93f4fd57d787c24db0e01ca304e5a5c4dd50cf8bdbf491e57c",
	}, { // Vector 5
		"2718281828459045235360287471352631415926535897932384626433832795",
		1,
		"27a7479befa1d476489f308cd4cfa6e2a96e4bbe3208ff25287dd3819616e89cc78cf7f5e543445f8333d8fa7f56000005279fa5d8b5e4ad40e736ddb4d35412328063fd2aab53e5ea1e0a9f332500a5df9487d07a5c92cc512c8866c7e860ce93fdf166a24912b422976146ae20ce846bb7dc9ba94a767aaef20c0d61ad02655ea92dc4c4e41a8952c651d33174be51a10c421110e6d81588ede82103a252d8a750e8768defffed9122810aaeb99f9172af82b604dc4b8e51bcb08235a6f4341332e4ca60482a4ba1a03b3e65008fc5da76b70bf1690db4eae29c5f1badd03c5ccf2a55d705ddcd86d449511ceb7ec30bf12b1fa35b913f9f747a8afd1b130e94bff94effd01a91735ca1726acd0b197c4e5b03393697e126826fb6bbde8ecc1e08298516e2c9ed03ff3c1b7860f6de76d4cecd94c8119855ef5297ca67e9f3e7ff72b1e99785ca0a7e7720c5b36dc6d72cac9574c8cbbc2f801e23e56fd344b07f22154beba0f08ce8891e643ed995c94d9a69c9f1b5f499027a78572aeebd74d20cc39881c213ee770b1010e4bea718846977ae119f7a023ab58cca0ad752afe656bb3c17256a9f6e9bf19fdd5a38fc82bbe872c5539edb609ef4f79c203ebb140f2e583cb2ad15b4aa5b655016a8449277dbd477ef2c8d6c017db738b18deb4a427d1923ce3ff262735779a418f20a282df920147beabe421ee5319d0568",
		"0e36ba273dd121afc77e0d8c00aa4a662b21f363470d333f2fe2ddcbcc51ecd523022f5fa7970062800cd3859cacead369263681543db431f3844a3638e837cf025cecc3b778e14ac1fd02bb684d0e3cc3d05758cf4b3827bae92f9f09a45487e0a830154a4206a14c4077bcc928e6039b78cdf8f915236c5a4efc21a0ba7173232cef6f18f8b53be5e1eb37282bed31a24f322cf1bba02dfd2583ce216a73726915116fd8ce46d58aa562b5a5d88076792d6e35cba40552db6a19776eaf255c3fc927adc41cb83a83884f98176267f37e543ce34fa32960d1d05aa05ff04103037a730175f1d59a32b64f308925fc9fa9c60421b4ab438e14504227cba20c8c06b508554fb02e52b92a1cd0a8e386511bc4c2fb62998d0ac5d9e7614080a10039b8cddf24a644b3e0aa02bb5d6c0897a84bfe0d12690cbd9fb92fd39b5b9504deeeaab0c5b9839b6283b87abe6439d28f0afb0508104fd4db9fd6e0301c6a488e76fd2a4801d2b7df57e0179506e9a8dbd7312be3922ea4e7339227061485452296dabc3b0f178a2e4ba012bbb6e836dec5d25abaa0f399ca622c5f075dfae7b2ffef4e396cd74b9bc3aeb7c212a5fd5c42b73fcf92e1f4ca458bb50e7257c4ffea253f30f7eaf9a6762ce15177f55ba250a4293d6ecdbd2e9a80c942b38dbdbd74773245a7a7db6b91d1f6c74bd32b7a7a193a2d260d266b64dd19b959ae42",
	}, { // XTS-SM4-128 applied for a data unit that is not a multiple of 16 bytes, but should be a complte byte
		"c46acc2e7e013cb71cdbf750cf76b000249fbf4fb6cd17607773c23ffa2c4330",
		94,
		"7e9c2289cba460e470222953439cdaa892a5433d4dab2a3f67",
		"c3cf5445c64aa518f4abce2848faddfb4605d9fb66f1f12c0c",
	}, {
		"56ffcc9bbbdf413f0fc0f888f44b7493bb1925a39b8adf02d9009bb16db0a887",
		144,
		"9a839cc14363bafcfc0cc93b14f8e769d35b94cc98267438e3",
		"af027012c829206c32a31706999d046f10a83bcacbc5c96353",
	},
	{
		"7454a43b87b1cf0dec95032c22873be3cace3bb795568854c1a008c07c5813f3",
		108,
		"41088fa15195b2733fe824d2c1fdc8306080863945fb2a73cf",
		"614ee9311a53791889338eb2f66fedff7dc15126349bed1465",
	},
}

func fromHex(s string) []byte {
	ret, err := hex.DecodeString(s)
	if err != nil {
		panic("xts: invalid hex in test")
	}
	return ret
}

func TestXTS(t *testing.T) {
	for i, test := range xtsTestVectors {
		key := fromHex(test.key)

		encrypter, err := cipher.NewXTSEncrypterWithSector(sm4.NewCipher, key[:len(key)/2], key[len(key)/2:], test.sector)
		if err != nil {
			t.Errorf("#%d: failed to create encrypter: %s", i, err)
			continue
		}
		decrypter, err := cipher.NewXTSDecrypterWithSector(sm4.NewCipher, key[:len(key)/2], key[len(key)/2:], test.sector)
		if err != nil {
			t.Errorf("#%d: failed to create decrypter: %s", i, err)
			continue
		}
		plaintext := fromHex(test.plaintext)
		ciphertext := make([]byte, len(plaintext))

		encrypter.CryptBlocks(ciphertext, plaintext)
		expectedCiphertext := fromHex(test.ciphertext)
		if !bytes.Equal(ciphertext, expectedCiphertext) {
			t.Errorf("#%d: encrypted failed, got: %x, want: %x", i, ciphertext, expectedCiphertext)
			continue
		}

		decrypted := make([]byte, len(ciphertext))
		decrypter.CryptBlocks(decrypted, ciphertext)
		if !bytes.Equal(decrypted, plaintext) {
			t.Errorf("#%d: decryption failed, got: %x, want: %x", i, decrypted, plaintext)
		}
	}
}

// Test data is from GB/T 17964-2021 B.7
var xtsGBTestVectors = []struct {
	key        string
	tweak      string
	plaintext  string
	ciphertext string
}{
	{
		"2B7E151628AED2A6ABF7158809CF4F3C000102030405060708090A0B0C0D0E0F",
		"F0F1F2F3F4F5F6F7F8F9FAFBFCFDFEFF",
		"6BC1BEE22E409F96E93D7E117393172AAE2D8A571E03AC9C9EB76FAC45AF8E5130C81C46A35CE411E5FBC1191A0A52EFF69F2445DF4F9B17",
		"E9538251C71D7B80BBE4483FEF497BD12C5C581BD6242FC51E08964FB4F60FDB0BA42F63499279213D318D2C11F6886E903BE7F93A1B3479",
	},
}

func TestXTS_GB(t *testing.T) {
	for i, test := range xtsGBTestVectors {
		key := fromHex(test.key)
		tweak := fromHex(test.tweak)
		encrypter, err := cipher.NewGBXTSEncrypter(sm4.NewCipher, key[:len(key)/2], key[len(key)/2:], tweak)
		if err != nil {
			t.Errorf("#%d: failed to create encrypter: %s", i, err)
			continue
		}
		decrypter, err := cipher.NewGBXTSDecrypter(sm4.NewCipher, key[:len(key)/2], key[len(key)/2:], tweak)
		if err != nil {
			t.Errorf("#%d: failed to create decrypter: %s", i, err)
			continue
		}
		plaintext := fromHex(test.plaintext)
		ciphertext := make([]byte, len(plaintext))

		encrypter.CryptBlocks(ciphertext, plaintext)
		expectedCiphertext := fromHex(test.ciphertext)
		if !bytes.Equal(ciphertext, expectedCiphertext) {
			t.Errorf("#%d: encrypted failed, got: %x, want: %x", i, ciphertext, expectedCiphertext)
			continue
		}

		decrypted := make([]byte, len(ciphertext))
		decrypter.CryptBlocks(decrypted, ciphertext)
		if !bytes.Equal(decrypted, plaintext) {
			t.Errorf("#%d: decryption failed, got: %x, want: %x", i, decrypted, plaintext)
		}
	}
}
