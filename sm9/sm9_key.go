package sm9

import (
	"encoding/asn1"
	"encoding/pem"

	"errors"
	"io"
	"math/big"
	"sync"

	"github.com/emmansun/gmsm/sm9/bn256"
	"golang.org/x/crypto/cryptobyte"
	cryptobyte_asn1 "golang.org/x/crypto/cryptobyte/asn1"
)

// SignMasterPrivateKey master private key for sign, generated by KGC
type SignMasterPrivateKey struct {
	SignMasterPublicKey          // master public key
	D                   *big.Int // master private key
}

// SignMasterPublicKey master public key for sign, generated by KGC
type SignMasterPublicKey struct {
	MasterPublicKey *bn256.G2 // master public key
	pairOnce        sync.Once
	basePoint       *bn256.GT // the result of Pair(Gen1, pub.MasterPublicKey)
	tableGenOnce    sync.Once
	table           *[32 * 2]bn256.GTFieldTable // precomputed basePoint^n
}

// SignPrivateKey user private key for sign, generated by KGC
type SignPrivateKey struct {
	PrivateKey          *bn256.G1 // user private key
	SignMasterPublicKey           // master public key
}

// EncryptMasterPrivateKey master private key for encryption, generated by KGC
type EncryptMasterPrivateKey struct {
	EncryptMasterPublicKey          // master public key
	D                      *big.Int // master private key
}

// EncryptMasterPublicKey master private key for encryption, generated by KGC
type EncryptMasterPublicKey struct {
	MasterPublicKey *bn256.G1 // public key
	pairOnce        sync.Once
	basePoint       *bn256.GT // the result of Pair(pub.MasterPublicKey, Gen2)
	tableGenOnce    sync.Once
	table           *[32 * 2]bn256.GTFieldTable // precomputed basePoint^n
}

// EncryptPrivateKey user private key for encryption, generated by KGC
type EncryptPrivateKey struct {
	PrivateKey             *bn256.G2 // user private key
	EncryptMasterPublicKey           // master public key
}

// GenerateSignMasterKey generates a master public and private key pair for DSA usage.
func GenerateSignMasterKey(rand io.Reader) (*SignMasterPrivateKey, error) {
	k, err := randFieldElement(rand)
	if err != nil {
		return nil, err
	}

	priv := new(SignMasterPrivateKey)
	priv.D = k
	priv.MasterPublicKey = new(bn256.G2).ScalarBaseMult(k)
	return priv, nil
}

// MarshalASN1 marshal sign master private key to asn.1 format data according
// SM9 cryptographic algorithm application specification
func (master *SignMasterPrivateKey) MarshalASN1() ([]byte, error) {
	var b cryptobyte.Builder
	b.AddASN1BigInt(master.D)
	return b.Bytes()
}

// UnmarshalASN1 unmarsal der data to sign master private key
func (master *SignMasterPrivateKey) UnmarshalASN1(der []byte) error {
	input := cryptobyte.String(der)
	d := &big.Int{}
	if !input.ReadASN1Integer(d) || !input.Empty() {
		return errors.New("sm9: invalid sign master key asn1 data")
	}
	master.D = d
	master.MasterPublicKey = new(bn256.G2).ScalarBaseMult(d)
	return nil
}

// GenerateUserKey generate an user dsa key.
func (master *SignMasterPrivateKey) GenerateUserKey(uid []byte, hid byte) (*SignPrivateKey, error) {
	var id []byte
	id = append(id, uid...)
	id = append(id, hid)

	t1 := hashH1(id)
	t1.Add(t1, master.D)
	if t1.Sign() == 0 {
		return nil, errors.New("sm9: need to re-generate sign master private key")
	}
	t1 = fermatInverse(t1, bn256.Order)
	t2 := new(big.Int).Mul(t1, master.D)
	t2.Mod(t2, bn256.Order)

	priv := new(SignPrivateKey)
	priv.SignMasterPublicKey = master.SignMasterPublicKey
	priv.PrivateKey = new(bn256.G1).ScalarBaseMult(t2)

	return priv, nil
}

// Public returns the public key corresponding to priv.
func (master *SignMasterPrivateKey) Public() *SignMasterPublicKey {
	return &master.SignMasterPublicKey
}

// pair generate the basepoint once
func (pub *SignMasterPublicKey) pair() *bn256.GT {
	pub.pairOnce.Do(func() {
		pub.basePoint = bn256.Pair(bn256.Gen1, pub.MasterPublicKey)
	})
	return pub.basePoint
}

func (pub *SignMasterPublicKey) generatorTable() *[32 * 2]bn256.GTFieldTable {
	pub.tableGenOnce.Do(func() {
		pub.table = bn256.GenerateGTFieldTable(pub.pair())
	})
	return pub.table
}

// ScalarBaseMult compute basepoint^r with precomputed table
// The base point = pair(Gen1, <master public key>)
func (pub *SignMasterPublicKey) ScalarBaseMult(r *big.Int) *bn256.GT {
	tables := pub.generatorTable()
	return bn256.ScalarBaseMultGT(tables, r)
}

// GenerateUserPublicKey generate user sign public key
func (pub *SignMasterPublicKey) GenerateUserPublicKey(uid []byte, hid byte) *bn256.G2 {
	var buffer []byte
	buffer = append(buffer, uid...)
	buffer = append(buffer, hid)
	h1 := hashH1(buffer)
	p := new(bn256.G2).ScalarBaseMult(h1)
	p.Add(p, pub.MasterPublicKey)
	return p
}

// MarshalASN1 marshal sign master public key to asn.1 format data according
// SM9 cryptographic algorithm application specification
func (pub *SignMasterPublicKey) MarshalASN1() ([]byte, error) {
	var b cryptobyte.Builder
	b.AddASN1BitString(pub.MasterPublicKey.MarshalUncompressed())
	return b.Bytes()
}

// MarshalCompressedASN1 marshal sign master public key to asn.1 format data according
// SM9 cryptographic algorithm application specification, the curve point is in compressed form.
func (pub *SignMasterPublicKey) MarshalCompressedASN1() ([]byte, error) {
	var b cryptobyte.Builder
	b.AddASN1BitString(pub.MasterPublicKey.MarshalCompressed())
	return b.Bytes()
}

func unmarshalG2(bytes []byte) (*bn256.G2, error) {
	g2 := new(bn256.G2)
	switch bytes[0] {
	case 4:
		_, err := g2.Unmarshal(bytes[1:])
		if err != nil {
			return nil, err
		}
	case 2, 3:
		_, err := g2.UnmarshalCompressed(bytes)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("sm9: invalid point identity byte")
	}
	return g2, nil
}

// UnmarshalRaw unmarsal raw bytes data to sign master public key
func (pub *SignMasterPublicKey) UnmarshalRaw(bytes []byte) error {
	g2, err := unmarshalG2(bytes)
	if err != nil {
		return err
	}
	pub.MasterPublicKey = g2
	return nil
}

// UnmarshalASN1 unmarsal der data to sign master public key
func (pub *SignMasterPublicKey) UnmarshalASN1(der []byte) error {
	var bytes []byte
	input := cryptobyte.String(der)
	if !input.ReadASN1BitStringAsBytes(&bytes) || !input.Empty() {
		return errors.New("sm9: invalid sign master public key asn1 data")
	}
	return pub.UnmarshalRaw(bytes)
}

type publicKeyInfo struct {
	Raw       asn1.RawContent
	PublicKey asn1.BitString
}

// ParseFromPEM just for GMSSL, there are no Algorithm pkix.AlgorithmIdentifier
func (pub *SignMasterPublicKey) ParseFromPEM(data []byte) error {
	block, _ := pem.Decode([]byte(data))
	if block == nil {
		return errors.New("failed to parse PEM block")
	}

	var pki publicKeyInfo
	if rest, err := asn1.Unmarshal(block.Bytes, &pki); err != nil {
		return err
	} else if len(rest) != 0 {
		return errors.New("trailing data after ASN.1 of public-key")
	}
	der := cryptobyte.String(pki.PublicKey.RightAlign())
	return pub.UnmarshalRaw(der)
}

// MasterPublic returns the master public key corresponding to priv.
func (priv *SignPrivateKey) MasterPublic() *SignMasterPublicKey {
	return &priv.SignMasterPublicKey
}

// SetMasterPublicKey bind the sign master public key to it.
func (priv *SignPrivateKey) SetMasterPublicKey(pub *SignMasterPublicKey) {
	if priv.SignMasterPublicKey.MasterPublicKey == nil {
		priv.SignMasterPublicKey = *pub
	}
}

// MarshalASN1 marshal sign user private key to asn.1 format data according
// SM9 cryptographic algorithm application specification
func (priv *SignPrivateKey) MarshalASN1() ([]byte, error) {
	var b cryptobyte.Builder
	b.AddASN1BitString(priv.PrivateKey.MarshalUncompressed())
	return b.Bytes()
}

// MarshalCompressedASN1 marshal sign user private key to asn.1 format data according
// SM9 cryptographic algorithm application specification, the curve point is in compressed form.
func (priv *SignPrivateKey) MarshalCompressedASN1() ([]byte, error) {
	var b cryptobyte.Builder
	b.AddASN1BitString(priv.PrivateKey.MarshalCompressed())
	return b.Bytes()
}

func unmarshalG1(bytes []byte) (*bn256.G1, error) {
	g := new(bn256.G1)
	switch bytes[0] {
	case 4:
		_, err := g.Unmarshal(bytes[1:])
		if err != nil {
			return nil, err
		}
	case 2, 3:
		_, err := g.UnmarshalCompressed(bytes)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("sm9: invalid point identity byte")
	}
	return g, nil
}

// UnmarshalRaw unmarsal raw bytes data to sign user private key
// Note, priv's SignMasterPublicKey should be handled separately.
func (priv *SignPrivateKey) UnmarshalRaw(bytes []byte) error {
	g, err := unmarshalG1(bytes)
	if err != nil {
		return err
	}
	priv.PrivateKey = g
	return nil
}

// UnmarshalASN1 unmarsal der data to sign user private key
// Note, priv's SignMasterPublicKey should be handled separately.
func (priv *SignPrivateKey) UnmarshalASN1(der []byte) error {
	var bytes []byte
	var pubBytes []byte
	var inner cryptobyte.String
	input := cryptobyte.String(der)
	if der[0] == 0x30 {
		if !input.ReadASN1(&inner, cryptobyte_asn1.SEQUENCE) ||
			!input.Empty() ||
			!inner.ReadASN1BitStringAsBytes(&bytes) {
			return errors.New("sm9: invalid sign user private key asn1 data")
		}
		if !inner.Empty() && (!inner.ReadASN1BitStringAsBytes(&pubBytes) || !inner.Empty()) {
			return errors.New("sm9: invalid sign master public key asn1 data")
		}
	} else if !input.ReadASN1BitStringAsBytes(&bytes) || !input.Empty() {
		return errors.New("sm9: invalid sign user private key asn1 data")
	}
	err := priv.UnmarshalRaw(bytes)
	if err != nil {
		return err
	}
	if len(pubBytes) > 0 {
		masterPK := new(SignMasterPublicKey)
		err = masterPK.UnmarshalRaw(pubBytes)
		if err != nil {
			return err
		}
		priv.SetMasterPublicKey(masterPK)
	}
	return nil
}

// GenerateEncryptMasterKey generates a master public and private key pair for encryption usage.
func GenerateEncryptMasterKey(rand io.Reader) (*EncryptMasterPrivateKey, error) {
	k, err := randFieldElement(rand)
	if err != nil {
		return nil, err
	}

	priv := new(EncryptMasterPrivateKey)
	priv.D = k
	priv.MasterPublicKey = new(bn256.G1).ScalarBaseMult(k)
	return priv, nil
}

// GenerateUserKey generate an user key for encryption.
func (master *EncryptMasterPrivateKey) GenerateUserKey(uid []byte, hid byte) (*EncryptPrivateKey, error) {
	var id []byte
	id = append(id, uid...)
	id = append(id, hid)

	t1 := hashH1(id)
	t1.Add(t1, master.D)
	if t1.Sign() == 0 {
		return nil, errors.New("sm9: need to re-generate encrypt master private key")
	}
	t1 = fermatInverse(t1, bn256.Order)
	t2 := new(big.Int).Mul(t1, master.D)
	t2.Mod(t2, bn256.Order)

	priv := new(EncryptPrivateKey)
	priv.EncryptMasterPublicKey = master.EncryptMasterPublicKey
	priv.PrivateKey = new(bn256.G2).ScalarBaseMult(t2)

	return priv, nil
}

// Public returns the public key corresponding to priv.
func (master *EncryptMasterPrivateKey) Public() *EncryptMasterPublicKey {
	return &master.EncryptMasterPublicKey
}

// MarshalASN1 marshal encrypt master private key to asn.1 format data according
// SM9 cryptographic algorithm application specification
func (master *EncryptMasterPrivateKey) MarshalASN1() ([]byte, error) {
	var b cryptobyte.Builder
	b.AddASN1BigInt(master.D)
	return b.Bytes()
}

// UnmarshalASN1 unmarsal der data to encrpt master private key
func (master *EncryptMasterPrivateKey) UnmarshalASN1(der []byte) error {
	input := cryptobyte.String(der)
	d := &big.Int{}
	if !input.ReadASN1Integer(d) || !input.Empty() {
		return errors.New("sm9: invalid encrpt master key asn1 data")
	}
	master.D = d
	master.MasterPublicKey = new(bn256.G1).ScalarBaseMult(d)
	return nil
}

// pair generate the basepoint once
func (pub *EncryptMasterPublicKey) pair() *bn256.GT {
	pub.pairOnce.Do(func() {
		pub.basePoint = bn256.Pair(pub.MasterPublicKey, bn256.Gen2)
	})
	return pub.basePoint
}

func (pub *EncryptMasterPublicKey) generatorTable() *[32 * 2]bn256.GTFieldTable {
	pub.tableGenOnce.Do(func() {
		pub.table = bn256.GenerateGTFieldTable(pub.pair())
	})
	return pub.table
}

// ScalarBaseMult compute basepoint^r with precomputed table.
// The base point = pair(<master public key>, Gen2)
func (pub *EncryptMasterPublicKey) ScalarBaseMult(r *big.Int) *bn256.GT {
	tables := pub.generatorTable()
	return bn256.ScalarBaseMultGT(tables, r)
}

// GenerateUserPublicKey generate user encrypt public key
func (pub *EncryptMasterPublicKey) GenerateUserPublicKey(uid []byte, hid byte) *bn256.G1 {
	var buffer []byte
	buffer = append(buffer, uid...)
	buffer = append(buffer, hid)
	h1 := hashH1(buffer)
	p := new(bn256.G1).ScalarBaseMult(h1)
	p.Add(p, pub.MasterPublicKey)
	return p
}

// MarshalASN1 marshal encrypt master public key to asn.1 format data according
// SM9 cryptographic algorithm application specification
func (pub *EncryptMasterPublicKey) MarshalASN1() ([]byte, error) {
	var b cryptobyte.Builder
	b.AddASN1BitString(pub.MasterPublicKey.MarshalUncompressed())
	return b.Bytes()
}

// MarshalCompressedASN1 marshal encrypt master public key to asn.1 format data according
// SM9 cryptographic algorithm application specification, the curve point is in compressed form.
func (pub *EncryptMasterPublicKey) MarshalCompressedASN1() ([]byte, error) {
	var b cryptobyte.Builder
	b.AddASN1BitString(pub.MasterPublicKey.MarshalCompressed())
	return b.Bytes()
}

// UnmarshalRaw unmarsal raw bytes data to encrypt master public key
func (pub *EncryptMasterPublicKey) UnmarshalRaw(bytes []byte) error {
	g, err := unmarshalG1(bytes)
	if err != nil {
		return err
	}
	pub.MasterPublicKey = g
	return nil
}

// ParseFromPEM just for GMSSL, there are no Algorithm pkix.AlgorithmIdentifier
func (pub *EncryptMasterPublicKey) ParseFromPEM(data []byte) error {
	block, _ := pem.Decode([]byte(data))
	if block == nil {
		return errors.New("failed to parse PEM block")
	}

	var pki publicKeyInfo
	if rest, err := asn1.Unmarshal(block.Bytes, &pki); err != nil {
		return err
	} else if len(rest) != 0 {
		return errors.New("trailing data after ASN.1 of public-key")
	}
	der := cryptobyte.String(pki.PublicKey.RightAlign())
	return pub.UnmarshalRaw(der)
}

// UnmarshalASN1 unmarsal der data to encrypt master public key
func (pub *EncryptMasterPublicKey) UnmarshalASN1(der []byte) error {
	var bytes []byte
	input := cryptobyte.String(der)
	if !input.ReadASN1BitStringAsBytes(&bytes) || !input.Empty() {
		return errors.New("sm9: invalid encrypt master public key asn1 data")
	}
	return pub.UnmarshalRaw(bytes)
}

// MasterPublic returns the master public key corresponding to priv.
func (priv *EncryptPrivateKey) MasterPublic() *EncryptMasterPublicKey {
	return &priv.EncryptMasterPublicKey
}

// SetMasterPublicKey bind the encrypt master public key to it.
func (priv *EncryptPrivateKey) SetMasterPublicKey(pub *EncryptMasterPublicKey) {
	if priv.EncryptMasterPublicKey.MasterPublicKey == nil {
		priv.EncryptMasterPublicKey = *pub
	}
}

// MarshalASN1 marshal encrypt user private key to asn.1 format data according
// SM9 cryptographic algorithm application specification
func (priv *EncryptPrivateKey) MarshalASN1() ([]byte, error) {
	var b cryptobyte.Builder
	b.AddASN1BitString(priv.PrivateKey.MarshalUncompressed())
	return b.Bytes()
}

// MarshalCompressedASN1 marshal encrypt user private key to asn.1 format data according
// SM9 cryptographic algorithm application specification, the curve point is in compressed form.
func (priv *EncryptPrivateKey) MarshalCompressedASN1() ([]byte, error) {
	var b cryptobyte.Builder
	b.AddASN1BitString(priv.PrivateKey.MarshalCompressed())
	return b.Bytes()
}

// UnmarshalRaw unmarsal raw bytes data to encrypt user private key
// Note, priv's EncryptMasterPublicKey should be handled separately.
func (priv *EncryptPrivateKey) UnmarshalRaw(bytes []byte) error {
	g, err := unmarshalG2(bytes)
	if err != nil {
		return err
	}
	priv.PrivateKey = g
	return nil
}

// UnmarshalASN1 unmarsal der data to encrypt user private key
// Note, priv's EncryptMasterPublicKey should be handled separately.
func (priv *EncryptPrivateKey) UnmarshalASN1(der []byte) error {
	var bytes []byte
	var pubBytes []byte
	var inner cryptobyte.String
	input := cryptobyte.String(der)
	if der[0] == 0x30 {
		if !input.ReadASN1(&inner, cryptobyte_asn1.SEQUENCE) ||
			!input.Empty() ||
			!inner.ReadASN1BitStringAsBytes(&bytes) {
			return errors.New("sm9: invalid encrypt user private key asn1 data")
		}
		if !inner.Empty() && (!inner.ReadASN1BitStringAsBytes(&pubBytes) || !inner.Empty()) {
			return errors.New("sm9: invalid encrypt master public key asn1 data")
		}
	} else if !input.ReadASN1BitStringAsBytes(&bytes) || !input.Empty() {
		return errors.New("sm9: invalid encrypt user private key asn1 data")
	}
	err := priv.UnmarshalRaw(bytes)
	if err != nil {
		return err
	}
	if len(pubBytes) > 0 {
		masterPK := new(EncryptMasterPublicKey)
		err = masterPK.UnmarshalRaw(pubBytes)
		if err != nil {
			return err
		}
		priv.SetMasterPublicKey(masterPK)
	}
	return nil
}

// fermatInverse calculates the inverse of k in GF(P) using Fermat's method
// (exponentiation modulo P - 2, per Euler's theorem). This has better
// constant-time properties than Euclid's method (implemented in
// math/big.Int.ModInverse and FIPS 186-4, Appendix C.1) although math/big
// itself isn't strictly constant-time so it's not perfect.
func fermatInverse(k, N *big.Int) *big.Int {
	two := big.NewInt(2)
	nMinus2 := new(big.Int).Sub(N, two)
	return new(big.Int).Exp(k, nMinus2, N)
}
