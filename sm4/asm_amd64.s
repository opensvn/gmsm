// This SM4 implementation referenced https://github.com/mjosaarinen/sm4ni/blob/master/sm4ni.c
#include "textflag.h"

#define x X0
#define y X1
#define t0 X2
#define t1 X3
#define t2 X4
#define t3 X5

#define XTMP6 X6
#define XTMP7 X7

// shuffle byte order from LE to BE
DATA flip_mask<>+0x00(SB)/8, $0x0405060700010203
DATA flip_mask<>+0x08(SB)/8, $0x0c0d0e0f08090a0b
GLOBL flip_mask<>(SB), RODATA, $16

//nibble mask
DATA nibble_mask<>+0x00(SB)/8, $0x0F0F0F0F0F0F0F0F
DATA nibble_mask<>+0x08(SB)/8, $0x0F0F0F0F0F0F0F0F
GLOBL nibble_mask<>(SB), RODATA, $16

// inverse shift rows
DATA inverse_shift_rows<>+0x00(SB)/8, $0x0B0E0104070A0D00
DATA inverse_shift_rows<>+0x08(SB)/8, $0x0306090C0F020508 
GLOBL inverse_shift_rows<>(SB), RODATA, $16

// Affine transform 1 (low and high hibbles)
DATA m1_low<>+0x00(SB)/8, $0x9197E2E474720701
DATA m1_low<>+0x08(SB)/8, $0xC7C1B4B222245157
GLOBL m1_low<>(SB), RODATA, $16

DATA m1_high<>+0x00(SB)/8, $0xE240AB09EB49A200
DATA m1_high<>+0x08(SB)/8, $0xF052B91BF95BB012  
GLOBL m1_high<>(SB), RODATA, $16

// Affine transform 2 (low and high hibbles)
DATA m2_low<>+0x00(SB)/8, $0x5B67F2CEA19D0834
DATA m2_low<>+0x08(SB)/8, $0xEDD14478172BBE82
GLOBL m2_low<>(SB), RODATA, $16

DATA m2_high<>+0x00(SB)/8, $0xAE7201DD73AFDC00
DATA m2_high<>+0x08(SB)/8, $0x11CDBE62CC1063BF
GLOBL m2_high<>(SB), RODATA, $16

// left rotations of 32-bit words by 8-bit increments
DATA r08_mask<>+0x00(SB)/8, $0x0605040702010003
DATA r08_mask<>+0x08(SB)/8, $0x0E0D0C0F0A09080B  
GLOBL r08_mask<>(SB), RODATA, $16

DATA r16_mask<>+0x00(SB)/8, $0x0504070601000302
DATA r16_mask<>+0x08(SB)/8, $0x0D0C0F0E09080B0A   
GLOBL r16_mask<>(SB), RODATA, $16

DATA r24_mask<>+0x00(SB)/8, $0x0407060500030201
DATA r24_mask<>+0x08(SB)/8, $0x0C0F0E0D080B0A09  
GLOBL r24_mask<>(SB), RODATA, $16

DATA fk_mask<>+0x00(SB)/8, $0x56aa3350a3b1bac6
DATA fk_mask<>+0x08(SB)/8, $0xb27022dc677d9197
GLOBL fk_mask<>(SB), RODATA, $16

#define SM4_SBOX(x, y) \
  ;                                   \ //#############################  inner affine ############################//
  MOVOU x, XTMP6;                     \
  PAND nibble_mask<>(SB), XTMP6;      \ //y = _mm_and_si128(x, c0f); 
  MOVOU m1_low<>(SB), y;              \
  PSHUFB XTMP6, y;                    \ //y = _mm_shuffle_epi8(m1l, y);
  PSRLQ $4, x;                        \ //x = _mm_srli_epi64(x, 4); 
  PAND nibble_mask<>(SB), x;          \ //x = _mm_and_si128(x, c0f);
  MOVOU m1_high<>(SB), XTMP6;         \
  PSHUFB x, XTMP6;                    \ //x = _mm_shuffle_epi8(m1h, x);
  MOVOU  XTMP6, x;                    \ //x = _mm_shuffle_epi8(m1h, x);
  PXOR y, x;                          \ //x = _mm_shuffle_epi8(m1h, x) ^ y;
  ;                                   \ // inverse ShiftRows
  PSHUFB inverse_shift_rows<>(SB), x; \ //x = _mm_shuffle_epi8(x, shr); 
  AESENCLAST nibble_mask<>(SB), x;    \ // AESNI instruction
  ;                                   \ //#############################  outer affine ############################//
  MOVOU  x, XTMP6;                    \
  PANDN nibble_mask<>(SB), XTMP6;     \ //XTMP6 = _mm_andnot_si128(x, c0f);
  MOVOU m2_low<>(SB), y;              \ 
  PSHUFB XTMP6, y;                    \ //y = _mm_shuffle_epi8(m2l, XTMP6)
  PSRLQ $4, x;                        \ //x = _mm_srli_epi64(x, 4);
  PAND nibble_mask<>(SB), x;          \ //x = _mm_and_si128(x, c0f); 
  MOVOU m2_high<>(SB), XTMP6;         \
  PSHUFB x, XTMP6;                    \
  MOVOU  XTMP6, x;                    \ //x = _mm_shuffle_epi8(m2h, x)
  PXOR y, x;                          \ //x = _mm_shuffle_epi8(m2h, x) ^ y; 

#define SM4_TAO_L1(x, y)         \
  SM4_SBOX(x, y);                     \
  ;                                   \ //####################  4 parallel L1 linear transforms ##################//
  MOVOU x, y;                         \
  PSHUFB r08_mask<>(SB), y;           \ //y = _mm_shuffle_epi8(x, r08)
  PXOR x, y;                          \ //y = x xor _mm_shuffle_epi8(x, r08)
  MOVOU x, XTMP6;                     \
  PSHUFB r16_mask<>(SB), XTMP6;       \ 
  PXOR XTMP6, y;                      \ //y = x xor _mm_shuffle_epi8(x, r08) xor _mm_shuffle_epi8(x, r16)
  MOVOU y, XTMP6;                     \
  PSLLL $2, XTMP6;                    \
  PSRLL $30, y;                       \
  POR XTMP6, y;                       \ //y = _mm_slli_epi32(y, 2) ^ _mm_srli_epi32(y, 30);  
  MOVOU x, XTMP7;                     \
  PSHUFB r24_mask<>(SB), XTMP7;       \
  PXOR y, x;                          \ //x = x xor y
  PXOR XTMP7, x                         //x = x xor y xor _mm_shuffle_epi8(x, r24);

#define SM4_TAO_L2(x, y)         \
  SM4_SBOX(x, y);                     \
  ;                                   \ //####################  4 parallel L2 linear transforms ##################//
  MOVOU x, y;                         \
  MOVOU x, XTMP6;                     \
  PSLLL $13, XTMP6;                   \
  PSRLL $19, y;                       \
  POR XTMP6, y;                      \ //y = X roll 13  
  PSLLL $10, XTMP6;                   \
  MOVOU x, XTMP7;                     \
  PSRLL $9, XTMP7;                    \
  POR XTMP6, XTMP7;                  \ //XTMP7 = x roll 23
  PXOR XTMP7, y;                      \
  PXOR y, x                        

// func expandKeyAsm(key *byte, ck, enc, dec *uint32)
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
  PINSRD $0, 0(BX)(CX*1), x
  PXOR t1, x
  PXOR t2, x
  PXOR t3, x
  SM4_TAO_L2(x, y)
  PXOR x, t0
  PEXTRD $0, t0, R8
  MOVL R8, 0(DX)(CX*1)
  MOVL R8, 12(DI)(SI*1)

  PINSRD $0, 4(BX)(CX*1), x
  PXOR t0, x
  PXOR t2, x
  PXOR t3, x
  SM4_TAO_L2(x, y)
  PXOR x, t1  
  PEXTRD $0, t1, R8
  MOVL R8, 4(DX)(CX*1)
  MOVL R8, 8(DI)(SI*1)
  
  PINSRD $0, 8(BX)(CX*1), x
  PXOR t0, x
  PXOR t1, x
  PXOR t3, x
  SM4_TAO_L2(x, y)
  PXOR x, t2
  PEXTRD $0, t2, R8
  MOVL R8, 8(DX)(CX*1)
  MOVL R8, 4(DI)(SI*1)

  PINSRD $0, 12(BX)(CX*1), x
  PXOR t0, x
  PXOR t1, x
  PXOR t2, x
  SM4_TAO_L2(x, y)
  PXOR x, t3  
  PEXTRD $0, t3, R8
  MOVL R8, 12(DX)(CX*1)
  MOVL R8, 0(DI)(SI*1)

  ADDL $16, CX
  SUBL $16, SI
  CMPL CX, $4*32
  JB loop

expand_end:  
  RET 

// func encryptBlocksAsm(xk *uint32, dst, src *byte)
TEXT ·encryptBlocksAsm(SB),NOSPLIT,$0
  MOVQ xk+0(FP), AX
  MOVQ dst+8(FP), BX
  MOVQ src+16(FP), DX
  
  PINSRD $0, 0(DX), t0
  PINSRD $1, 16(DX), t0
  PINSRD $2, 32(DX), t0
  PINSRD $3, 48(DX), t0
  PSHUFB flip_mask<>(SB), t0

  PINSRD $0, 4(DX), t1
  PINSRD $1, 20(DX), t1
  PINSRD $2, 36(DX), t1
  PINSRD $3, 52(DX), t1
  PSHUFB flip_mask<>(SB), t1

  PINSRD $0, 8(DX), t2
  PINSRD $1, 24(DX), t2
  PINSRD $2, 40(DX), t2
  PINSRD $3, 56(DX), t2
  PSHUFB flip_mask<>(SB), t2

  PINSRD $0, 12(DX), t3
  PINSRD $1, 28(DX), t3
  PINSRD $2, 44(DX), t3
  PINSRD $3, 60(DX), t3
  PSHUFB flip_mask<>(SB), t3

  XORL CX, CX

loop:
  PINSRD $0, 0(AX)(CX*1), x
  PSHUFD $0, x, x
  PXOR t1, x
  PXOR t2, x
  PXOR t3, x
  SM4_TAO_L1(x, y)
  PXOR x, t0

  PINSRD $0, 4(AX)(CX*1), x
  PSHUFD $0, x, x
  PXOR t0, x
  PXOR t2, x
  PXOR t3, x
  SM4_TAO_L1(x, y)
  PXOR x, t1  

  PINSRD $0, 8(AX)(CX*1), x
  PSHUFD $0, x, x
  PXOR t0, x
  PXOR t1, x
  PXOR t3, x
  SM4_TAO_L1(x, y)
  PXOR x, t2

  PINSRD $0, 12(AX)(CX*1), x
  PSHUFD $0, x, x
  PXOR t0, x
  PXOR t1, x
  PXOR t2, x
  SM4_TAO_L1(x, y)
  PXOR x, t3  

  ADDL $16, CX
  CMPL CX, $4*32
  JB loop

  PSHUFB flip_mask<>(SB), t3
  PSHUFB flip_mask<>(SB), t2
  PSHUFB flip_mask<>(SB), t1
  PSHUFB flip_mask<>(SB), t0
  MOVUPS t3, 0(BX)
  MOVUPS t2, 16(BX)
  MOVUPS t1, 32(BX)
  MOVUPS t0, 48(BX)
  MOVL  4(BX), R8
  MOVL  8(BX), R9
  MOVL  12(BX), R10
  MOVL  16(BX), R11
  MOVL  32(BX), R12
  MOVL  48(BX), R13
  MOVL  R11, 4(BX)
  MOVL  R12, 8(BX)
  MOVL  R13, 12(BX)
  MOVL  R8, 16(BX)
  MOVL  R9, 32(BX)
  MOVL  R10, 48(BX)
  MOVL  24(BX), R8
  MOVL  28(BX), R9
  MOVL  36(BX), R10
  MOVL  52(BX), R11
  MOVL  R10, 24(BX)
  MOVL  R11, 28(BX)
  MOVL  R8, 36(BX)
  MOVL  R9, 52(BX)
  MOVL  44(BX), R8
  MOVL  56(BX), R9
  MOVL  R9, 44(BX)
  MOVL  R8, 56(BX)

done_sm4:
	RET

// func encryptBlockAsm(xk *uint32, dst, src *byte)
TEXT ·encryptBlockAsm(SB),NOSPLIT,$0
  MOVQ xk+0(FP), AX
  MOVQ dst+8(FP), BX
  MOVQ src+16(FP), DX
  
  PINSRD $0, 0(DX), t0
  PSHUFB flip_mask<>(SB), t0

  PINSRD $0, 4(DX), t1
  PSHUFB flip_mask<>(SB), t1

  PINSRD $0, 8(DX), t2
  PSHUFB flip_mask<>(SB), t2

  PINSRD $0, 12(DX), t3
  PSHUFB flip_mask<>(SB), t3

  XORL CX, CX

loop:
  PINSRD $0, 0(AX)(CX*1), x
  PXOR t1, x
  PXOR t2, x
  PXOR t3, x
  SM4_TAO_L1(x, y)
  PXOR x, t0

  PINSRD $0, 4(AX)(CX*1), x
  PXOR t0, x
  PXOR t2, x
  PXOR t3, x
  SM4_TAO_L1(x, y)
  PXOR x, t1  

  PINSRD $0, 8(AX)(CX*1), x
  PXOR t0, x
  PXOR t1, x
  PXOR t3, x
  SM4_TAO_L1(x, y)
  PXOR x, t2

  PINSRD $0, 12(AX)(CX*1), x
  PXOR t0, x
  PXOR t1, x
  PXOR t2, x
  SM4_TAO_L1(x, y)
  PXOR x, t3  

  ADDL $16, CX
  CMPL CX, $4*32
  JB loop

  PSHUFB flip_mask<>(SB), t3
  PSHUFB flip_mask<>(SB), t2
  PSHUFB flip_mask<>(SB), t1
  PSHUFB flip_mask<>(SB), t0
  MOVUPS t3, 0(BX)
  PEXTRD $0, t2, R8
  MOVL R8, 4(BX)
  PEXTRD $0, t1, R8
  MOVL R8, 8(BX)
  PEXTRD $0, t0, R8
  MOVL R8, 12(BX)
done_sm4:
	RET
