汇编开发
--------------

本文基于MAC介绍一些基本汇编开发，旨在帮助理解程序的底层执行原理.
具体的系统架构和操作系统都不重要，不同系统架构带来的无非是指令集和寄存器等差异，
最终程序的本质都是一样的.

字节定义

```
a byte as 8 bits, a word as 16 bits,
a double word as 32 bits, a quadword as 64 bits,
and a double quadword as 128 bits.
```

Intel数据存储方式

```
Intel stores bytes "little endian,"
meaning lower significant bytes are stored in lower memory addresses.
```

Little-endian - we can imagine memory as one large array. It contains bytes.
Each address stores one element of the memory “array”.
Each element is one byte. For example we have 4 bytes: AA 56 AB FF.
In little-endian the least significant byte has the smallest address:

```
0 FF
1 AB
2 56
3 AA
```

**Intel i7寄存器**

1. 64 bits

RAX, RBX, RCX, RDX, RBP, RSI, RDI, RSP, R8~R15. 共16个64bits寄存器;

对于前面8个：RAX, RBX, RCX, RDX, RBP, RSI, RDI, RSP

如果R改成E，比如RAX - EAX，就可以访问低32bits;
还可以把前面的R去掉，比如RAX - AX，访问低16bits; AL访问RAX的低8bits;
AX的高8bits可以用AH访问;

对于后面8个新寄存器：R8 - R15

如 R8, R8 (64bits), R8D (低32bits), R8W (低16bits), R8B (低8bits)

还有一个64bits的寄存器RIP，充当PC（程序计数器），存放下一条指令的地址

还有RSP存放栈顶; RFLAGS存放判断结果.

2. FPU(floating pointing unit)

The floating point unit (FPU) contains eight registers FPR0-FPR7

Single Instruction Multiple Data (SIMD) instructions execute a single command
on multiple pieces of data in parallel and are a common usage for assembly
routines. MMX and SSE commands (using the MMX and XMM registers respectively)
support SIMD operations, which perform an instruction on up to eight pieces of
data in parallel.
For example, eight bytes can be added to eight bytes in one instruction using MMX.

**NASM汇编**

本文基于MACOS 10.10，汇编器用`nasm`:

```
$ nasm -v
NASM version 2.12.01 compiled on Mar 23 2016
```

一定要保证`nasm`用比较新的版本，老版本可能不支持64bits汇编的编译.

本文中所有的例子都按照`e${order}`的格式编号，比如例1，文件就是`examples/e1.asm`.

通过如下命令来编译，检查编译环境是否满足要求：

```
// 生成Object文件
$ nasm -f macho64 -o play/e1.o examples/e1.asm
// 生成可执行文件，这里加上-e _main是因为nasm默认的入口函数的_start
$ ld -o play/e1 -e _main play/e1.o
// 运行可执行文件
$ ./play/e1

Hello, World!
```

也可以通过`build.sh`来编译和运行例子：

```
# 编译但不运行e1
$ sh build.sh e1 # e1也可以是e1.asm

Executable of e1 is located at /Users/yangyuqian/code/technical-articles/asm/play/e1

# 编译并运行e1
$ sh build.sh e1 exec

=============== Run example e1 ==============

+ /Users/yangyuqian/code/technical-articles/asm/play/e1
Hello, World!
```

# Hello World

例1: `examples/e1.asm` 是`Hello World`程序，向console输出一段字符

每个汇编程序都可以有3个`section`:

* `data`: 数据段, 用来存放常量
* `text`: 代码段
* `bss`: 用来存放程序中未初始化的全局变量的一块内存区域

首先看`data section`:

```
// 声明下面的代码都在数据段中
SECTION .data
// 定义变量
msg: db "Hello, World!", 0x0a
len: equ $-msg
```

说到常量的定义：

```
SECTION .data

// const1 = 100
const1: equ 100
```

那么上面的`db` `equ`是什么意思呢？这是`nasm`支持的 [pseudo-instructions](http://www.nasm.us/doc/nasmdoc3.html).

```
db    0x55                ; just the byte 0x55
db    0x55,0x56,0x57      ; three bytes in succession
db    'hello',13,10,'$'   ; so are string constants
dw    0x1234              ; 0x34 0x12
dw    'ab'                ; 0x61 0x62 (character constant)
```

所以上面的 `msg: db "xxx", 0x0a` 就是简单的定义了一个字符串常量.

用`$-data`可以获取`data`的数据长度，所以`len: equ $-msg`就定义了`len=len(msg)`.

```
message         db      'hello, world'
msglen          equ     $-message
```

还可以定义没有初始化的常量:

```
buffer:         resb    64              ; reserve 64 bytes
wordvar:        resw    1               ; reserve a word
realarray       resq    10              ; array of ten reals
ymmval:         resy    1               ; one YMM register
zmmvals:        resz    32              ; 32 ZMM registers
```

然后看`text section`, 正式进入代码逻辑：

```
SECTION .text
// 声明入口是_main
global _main

// 定义一个函数
kernel:
    syscall
    ret

// 程序入口函数
_main:
    // 表明当前syscall是写I/O操作
    mov rax,0x2000004
    // 表明当前写stdout
    mov rdi,1
    // 数据入口地址写入rsi
    mov rsi,msg
    // 数据长度写入rdx
    mov rdx,len
    call kernel

    // 表明当前系统调用是exit
    mov rax,0x2000001
    // exit的时候返回0，等价于`exit 0`
    mov rdi,0
    call kernel
```

在MAC中系统调用的标志比较有意思，需要把具体的值加上`0x2000000`.

例2 `examples/e2.asm`是一个更复杂一点的程序，从命令行获取2个参数，
判断参数是否是2个，输出2个参数相加的和.

从命令行获取`argc`

```
	pop	rcx
	cmp	rcx, 3
```

命令行的参数是一个个入栈的

```
argc
argv[0]: program name
...
argv[N - 1]
```

对于`Intel`的`Little-endian`处理器来说，栈是从小地址到大地址的，`rsp`存放了栈
顶地址

```
	;; skip argv[0] - program name
	add	rsp, 8
	;; argv[1]出栈，存放到rsi
	pop	rsi
  ...
  ;; argv[2]出栈，存放到rsi
	pop	rsi
  ...
```

在中间有时候使用`call`来调用子函数，也可以用`jmp`直接跳转一个程序段，
这是有区别的:

* `call`: 系统会发起一个中断，并将当前状态压栈，子函数中使用`RET`指令
可以恢复当前状态
* `jmp`: 不保存当前状态，直接跳转对应的程序段

本质上程序就是一堆指令的集合，这些指令按照某种顺序来执行就可以得到特定的结果.

在例2中有2个地方用了栈：

* 获取参数阶段，前面已经介绍过
* 打印输出的时候，把输出内容压栈，并把栈顶`rsp`传给`syscall`

把输入的2个参数转成数字：

```
	pop	rsi
	call	str_to_int
	mov	r10, rax

	pop	rsi
	call	str_to_int
	mov	r11, rax
```

这里写了一个子函数`str_to_int`

```
;; 输入[rsi]，输出rax
str_to_int:
	;; rax = 0
	xor	rax, rax
	mov	rcx,  10
next:
  ;; 判断最后一个字符
	cmp	[rsi], byte 0
	;; return int
	je	return_str
	;; NASM中给8 bit寄存器的别名
	mov	bl, [rsi]
	;; 单个ascii字符转成数字, 这里BL(8 bits)是RBX(64 bits)最低的8 bit
	sub	bl, 48
	;; rax = rax * rcx(10)
	mul	rcx
	;; ax = ax + digit
	add	rax, rbx
	;; 地址+1，获取下一个字符
	inc	rsi
	;; again
	jmp	next

return_str:
	ret
```

NASM中64 bits寄存器：

```
R0  R1  R2  R3  R4  R5  R6  R7  R8  R9  R10  R11  R12  R13  R14  R15
RAX RCX RDX RBX RSP RBP RSI RDI
```

注意在`Intel i7`中前面8个寄存器是`RAX ~ RDI`，但`NASM`里面给了别名`R0~R7`.

这里用到了8 bits的寄存器：

```
// 64 bits寄存器中最低的8 bits
// 下面一行寄存器名字是实际的名字，上面的是NASM中的别名

R0B R1B R2B R3B R4B R5B R6B R7B R8B R9B R10B R11B R12B R13B R14B R15B
AL  CL  DL  BL  SPL BPL SIL DIL
```


# References

[Intel i7 Assembly](https://software.intel.com/en-us/articles/introduction-to-x64-assembly)

[Say hello to x64 Assembly](http://0xax.blogspot.ca/2014/08/say-hello-to-x64-assembly-part-1.html)

[Examples](https://github.com/0xAX/asm)

[NASM Assembly](http://www.nasm.us/doc/nasmdoc3.html)

[Making system calls from Assembly in Mac OS X](https://filippo.io/making-system-calls-from-assembly-in-mac-os-x/)

[The Netwide Assembler: NASM](http://www.nasm.us/doc/nasmdo11.html)
