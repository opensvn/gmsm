// This SM4 implementation referenced https://github.com/mjosaarinen/sm4ni/blob/master/sm4ni.c
//go:build amd64 && !purego
// +build amd64,!purego

#include "textflag.h"

#define x X0
#define y X1
#define t0 X2
#define t1 X3
#define t2 X4
#define t3 X5

#define XTMP6 X6
#define XTMP7 X7

#include "aesni_macros_amd64.s"

// SM4 TAO L2 function, used for key expand
// parameters:
// -  x: 128 bits register as TAO_L1 input/output data
// -  y: 128 bits temp register
// -  tmp1: 128 bits temp register
// -  tmp2: 128 bits temp register
#define SM4_TAO_L2(x, y, tmp1, tmp2)    \
	SM4_SBOX(x, y, tmp1);              \
	;                                  \ //####################  4 parallel L2 linear transforms ##################//
	MOVOU x, y;                        \
	MOVOU x, tmp1;                     \
	PSLLL $13, tmp1;                   \
	PSRLL $19, y;                      \
	POR tmp1, y;                       \ //y = X roll 13  
	PSLLL $10, tmp1;                   \
	MOVOU x, tmp2;                     \
	PSRLL $9, tmp2;                    \
	POR tmp1, tmp2;                    \ //tmp2 = x roll 23
	PXOR tmp2, y;                      \
	PXOR y, x                        

// SM4 expand round function
// t0 ^= tao_l2(t1^t2^t3^ck) and store t0.S[0] to enc/dec
// parameters:
// - index: round key index immediate number
// -  x: 128 bits temp register
// -  y: 128 bits temp register
// - t0: 128 bits register for data
// - t1: 128 bits register for data
// - t2: 128 bits register for data
// - t3: 128 bits register for data
#define SM4_EXPANDKEY_ROUND(index, x, y, t0, t1, t2, t3) \
	PINSRD $0, (index * 4)(BX)(CX*1), x;                   \
	PXOR t1, x;                                            \
	PXOR t2, x;                                            \
	PXOR t3, x;                                            \
	SM4_TAO_L2(x, y, XTMP6, XTMP7);                        \
	PXOR x, t0;                                            \
	MOVL t0, R8;                                           \ // _mm_cvtsi128_si32
	MOVL R8, (index * 4)(DX)(CX*1);                        \
	MOVL R8, (12 - index * 4)(DI)(SI*1)

#define XDWORD0 Y4
#define XDWORD1 Y5
#define XDWORD2 Y6
#define XDWORD3 Y7

#define XWORD0 X4
#define XWORD1 X5
#define XWORD2 X6
#define XWORD3 X7

#define XDWTMP0 Y0
#define XDWTMP1 Y1
#define XDWTMP2 Y2

#define XWTMP0 X0
#define XWTMP1 X1
#define XWTMP2 X2

#define NIBBLE_MASK Y3
#define X_NIBBLE_MASK X3

#define BYTE_FLIP_MASK 	Y13 // mask to convert LE -> BE
#define X_BYTE_FLIP_MASK 	X13 // mask to convert LE -> BE

#define XDWORD Y8
#define YDWORD Y9

#define XWORD X8
#define YWORD X9

// func expandKeyAsm(key *byte, ck, enc, dec *uint32, inst int)
TEXT ·expandKeyAsm(SB),NOSPLIT,$0
	MOVQ key+0(FP), AX
	MOVQ  ck+8(FP), BX
	MOVQ  enc+16(FP), DX
	MOVQ  dec+24(FP), DI

	MOVUPS 0(AX), t0
	PSHUFB flip_mask<>(SB), t0
	PXOR fk_mask<>(SB), t0
	PSHUFD $1, t0, t1
	PSHUFD $2, t0, t2
	PSHUFD $3, t0, t3

	XORL CX, CX
	MOVL $112, SI

loop:
		SM4_EXPANDKEY_ROUND(0, x, y, t0, t1, t2, t3)
		SM4_EXPANDKEY_ROUND(1, x, y, t1, t2, t3, t0)
		SM4_EXPANDKEY_ROUND(2, x, y, t2, t3, t0, t1)
		SM4_EXPANDKEY_ROUND(3, x, y, t3, t0, t1, t2)

		ADDL $16, CX
		SUBL $16, SI
		CMPL CX, $4*32
		JB loop

expand_end:  
	RET 

// func encryptBlocksAsm(xk *uint32, dst, src []byte, inst int)
TEXT ·encryptBlocksAsm(SB),NOSPLIT,$0
	MOVQ xk+0(FP), AX
	MOVQ dst+8(FP), BX
	MOVQ src+32(FP), DX
	MOVQ src_len+40(FP), DI

	CMPB ·useAVX2(SB), $1
	JE   avx2

	CMPB ·useAVX(SB), $1
	JE   avx

non_avx2_start:
	MOVOU 0(DX), t0
	MOVOU 16(DX), t1
	MOVOU 32(DX), t2
	MOVOU 48(DX), t3
	PSHUFB flip_mask<>(SB), t0
	PSHUFB flip_mask<>(SB), t1
	PSHUFB flip_mask<>(SB), t2
	PSHUFB flip_mask<>(SB), t3
	SSE_TRANSPOSE_MATRIX(t0, t1, t2, t3, x, y)

	XORL CX, CX

loop:
		SM4_ROUND(0, AX, CX, x, y, XTMP6, t0, t1, t2, t3)
		SM4_ROUND(1, AX, CX, x, y, XTMP6, t1, t2, t3, t0)
		SM4_ROUND(2, AX, CX, x, y, XTMP6, t2, t3, t0, t1)
		SM4_ROUND(3, AX, CX, x, y, XTMP6, t3, t0, t1, t2)

		ADDL $16, CX
		CMPL CX, $4*32
		JB loop

	SSE_TRANSPOSE_MATRIX(t0, t1, t2, t3, x, y);  
	PSHUFB bswap_mask<>(SB), t3
	PSHUFB bswap_mask<>(SB), t2
	PSHUFB bswap_mask<>(SB), t1
	PSHUFB bswap_mask<>(SB), t0
	
	MOVOU t0, 0(BX)
	MOVOU t1, 16(BX)
	MOVOU t2, 32(BX)
	MOVOU t3, 48(BX)

done_sm4:
	RET

avx:
	VMOVDQU 0(DX), XWORD0
	VMOVDQU 16(DX), XWORD1
	VMOVDQU 32(DX), XWORD2
	VMOVDQU 48(DX), XWORD3

	VMOVDQU nibble_mask<>(SB), X_NIBBLE_MASK
	VMOVDQU flip_mask<>(SB), X_BYTE_FLIP_MASK

	VPSHUFB X_BYTE_FLIP_MASK, XWORD0, XWORD0
	VPSHUFB X_BYTE_FLIP_MASK, XWORD1, XWORD1
	VPSHUFB X_BYTE_FLIP_MASK, XWORD2, XWORD2
	VPSHUFB X_BYTE_FLIP_MASK, XWORD3, XWORD3

	// Transpose matrix 4 x 4 32bits word
	TRANSPOSE_MATRIX(XWORD0, XWORD1, XWORD2, XWORD3, XWTMP1, XWTMP2)

	XORL CX, CX

avx_loop:
		AVX_SM4_ROUND(0, AX, CX, XWORD, YWORD, XWTMP0, XWORD0, XWORD1, XWORD2, XWORD3)
		AVX_SM4_ROUND(1, AX, CX, XWORD, YWORD, XWTMP0, XWORD1, XWORD2, XWORD3, XWORD0)
		AVX_SM4_ROUND(2, AX, CX, XWORD, YWORD, XWTMP0, XWORD2, XWORD3, XWORD0, XWORD1)
		AVX_SM4_ROUND(3, AX, CX, XWORD, YWORD, XWTMP0, XWORD3, XWORD0, XWORD1, XWORD2)

		ADDL $16, CX
		CMPL CX, $4*32
		JB avx_loop

	// Transpose matrix 4 x 4 32bits word
	TRANSPOSE_MATRIX(XWORD0, XWORD1, XWORD2, XWORD3, XWTMP1, XWTMP2)

	VMOVDQU bswap_mask<>(SB), X_BYTE_FLIP_MASK
	VPSHUFB X_BYTE_FLIP_MASK, XWORD0, XWORD0
	VPSHUFB X_BYTE_FLIP_MASK, XWORD1, XWORD1
	VPSHUFB X_BYTE_FLIP_MASK, XWORD2, XWORD2
	VPSHUFB X_BYTE_FLIP_MASK, XWORD3, XWORD3

	VMOVDQU XWORD0, 0(BX)
	VMOVDQU XWORD1, 16(BX)
	VMOVDQU XWORD2, 32(BX)
	VMOVDQU XWORD3, 48(BX)

	RET

avx2:
	VBROADCASTI128 nibble_mask<>(SB), NIBBLE_MASK
	CMPQ DI, $64
	JBE   avx2_4blocks

avx2_8blocks:
	VMOVDQU 0(DX), XDWORD0
	VMOVDQU 32(DX), XDWORD1
	VMOVDQU 64(DX), XDWORD2
	VMOVDQU 96(DX), XDWORD3
	VBROADCASTI128 flip_mask<>(SB), BYTE_FLIP_MASK

	// Apply Byte Flip Mask: LE -> BE
	VPSHUFB BYTE_FLIP_MASK, XDWORD0, XDWORD0
	VPSHUFB BYTE_FLIP_MASK, XDWORD1, XDWORD1
	VPSHUFB BYTE_FLIP_MASK, XDWORD2, XDWORD2
	VPSHUFB BYTE_FLIP_MASK, XDWORD3, XDWORD3

	// Transpose matrix 4 x 4 32bits word
	TRANSPOSE_MATRIX(XDWORD0, XDWORD1, XDWORD2, XDWORD3, XDWTMP1, XDWTMP2)

	XORL CX, CX

avx2_loop:
		AVX2_SM4_ROUND(0, AX, CX, XDWORD, YDWORD, XWORD, YWORD, XDWTMP0, XDWORD0, XDWORD1, XDWORD2, XDWORD3)
		AVX2_SM4_ROUND(1, AX, CX, XDWORD, YDWORD, XWORD, YWORD, XDWTMP0, XDWORD1, XDWORD2, XDWORD3, XDWORD0)
		AVX2_SM4_ROUND(2, AX, CX, XDWORD, YDWORD, XWORD, YWORD, XDWTMP0, XDWORD2, XDWORD3, XDWORD0, XDWORD1)
		AVX2_SM4_ROUND(3, AX, CX, XDWORD, YDWORD, XWORD, YWORD, XDWTMP0, XDWORD3, XDWORD0, XDWORD1, XDWORD2)

		ADDL $16, CX
		CMPL CX, $4*32
		JB avx2_loop

	// Transpose matrix 4 x 4 32bits word
	TRANSPOSE_MATRIX(XDWORD0, XDWORD1, XDWORD2, XDWORD3, XDWTMP1, XDWTMP2)

	VBROADCASTI128 bswap_mask<>(SB), BYTE_FLIP_MASK
	VPSHUFB BYTE_FLIP_MASK, XDWORD0, XDWORD0
	VPSHUFB BYTE_FLIP_MASK, XDWORD1, XDWORD1
	VPSHUFB BYTE_FLIP_MASK, XDWORD2, XDWORD2
	VPSHUFB BYTE_FLIP_MASK, XDWORD3, XDWORD3
  
	VMOVDQU XDWORD0, 0(BX)
	VMOVDQU XDWORD1, 32(BX)
	VMOVDQU XDWORD2, 64(BX)
	VMOVDQU XDWORD3, 96(BX)
	JMP avx2_sm4_done

avx2_4blocks:
	VMOVDQU 0(DX), XWORD0
	VMOVDQU 16(DX), XWORD1
	VMOVDQU 32(DX), XWORD2
	VMOVDQU 48(DX), XWORD3

	VMOVDQU flip_mask<>(SB), X_BYTE_FLIP_MASK

	VPSHUFB X_BYTE_FLIP_MASK, XWORD0, XWORD0
	VPSHUFB X_BYTE_FLIP_MASK, XWORD1, XWORD1
	VPSHUFB X_BYTE_FLIP_MASK, XWORD2, XWORD2
	VPSHUFB X_BYTE_FLIP_MASK, XWORD3, XWORD3

	// Transpose matrix 4 x 4 32bits word
	TRANSPOSE_MATRIX(XWORD0, XWORD1, XWORD2, XWORD3, XWTMP1, XWTMP2)

	XORL CX, CX

avx2_4blocks_loop:
		AVX2_SM4_ROUND_4BLOCKS(0, AX, CX, XWORD, YWORD, XWTMP0, XWORD0, XWORD1, XWORD2, XWORD3)
		AVX2_SM4_ROUND_4BLOCKS(1, AX, CX, XWORD, YWORD, XWTMP0, XWORD1, XWORD2, XWORD3, XWORD0)
		AVX2_SM4_ROUND_4BLOCKS(2, AX, CX, XWORD, YWORD, XWTMP0, XWORD2, XWORD3, XWORD0, XWORD1)
		AVX2_SM4_ROUND_4BLOCKS(3, AX, CX, XWORD, YWORD, XWTMP0, XWORD3, XWORD0, XWORD1, XWORD2)

		ADDL $16, CX
		CMPL CX, $4*32
		JB avx2_4blocks_loop

	// Transpose matrix 4 x 4 32bits word
	TRANSPOSE_MATRIX(XWORD0, XWORD1, XWORD2, XWORD3, XWTMP1, XWTMP2)

	VMOVDQU bswap_mask<>(SB), X_BYTE_FLIP_MASK
	VPSHUFB X_BYTE_FLIP_MASK, XWORD0, XWORD0
	VPSHUFB X_BYTE_FLIP_MASK, XWORD1, XWORD1
	VPSHUFB X_BYTE_FLIP_MASK, XWORD2, XWORD2
	VPSHUFB X_BYTE_FLIP_MASK, XWORD3, XWORD3
  
	VMOVDQU XWORD0, 0(BX)
	VMOVDQU XWORD1, 16(BX)
	VMOVDQU XWORD2, 32(BX)
	VMOVDQU XWORD3, 48(BX)

avx2_sm4_done:
	VZEROUPPER
	RET

// func encryptBlockAsm(xk *uint32, dst, src *byte, inst int)
TEXT ·encryptBlockAsm(SB),NOSPLIT,$0
	MOVQ xk+0(FP), AX
	MOVQ dst+8(FP), BX
	MOVQ src+16(FP), DX
  
	MOVOU (DX), t0
	PSHUFB flip_mask<>(SB), t0
	PSHUFD $1, t0, t1
	PSHUFD $2, t0, t2
	PSHUFD $3, t0, t3

	XORL CX, CX

loop:
		SM4_SINGLE_ROUND(0, AX, CX, x, y, XTMP6, t0, t1, t2, t3)
		SM4_SINGLE_ROUND(1, AX, CX, x, y, XTMP6, t1, t2, t3, t0)
		SM4_SINGLE_ROUND(2, AX, CX, x, y, XTMP6, t2, t3, t0, t1)
		SM4_SINGLE_ROUND(3, AX, CX, x, y, XTMP6, t3, t0, t1, t2)

		ADDL $16, CX
		CMPL CX, $4*32
		JB loop

	PALIGNR $4, t3, t3
	PALIGNR $4, t3, t2
	PALIGNR $4, t2, t1
	PALIGNR $4, t1, t0
	PSHUFB flip_mask<>(SB), t0
	MOVOU t0, (BX)

done_sm4:
	RET
