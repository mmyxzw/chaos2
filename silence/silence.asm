; Chaos2 — silence module
; Assembly x86-64: decides when Chaos should NOT respond
; Input (stdin): "intent|strategy|intensity"
; Output: "1" = stay silent, "0" = speak

section .data
    out_silent  db "1", 10
    out_speak   db "0", 10
    buf         times 64 db 0

section .text
    global _start

_start:
    mov rax, 0
    mov rdi, 0
    mov rsi, buf
    mov rdx, 63
    syscall

    movzx rax, byte [buf]
    cmp rax, 'w'
    je .silent

    cmp rax, 'p'
    jne .speak

    mov rcx, 0
    mov rbx, 0
.scan:
    movzx rax, byte [buf + rcx]
    cmp rax, 0
    je .speak
    cmp rax, '|'
    jne .next
    inc rbx
    cmp rbx, 2
    je .check_intensity
.next:
    inc rcx
    jmp .scan

.check_intensity:
    inc rcx
    movzx rax, byte [buf + rcx]
    cmp rax, '0'
    je .silent
    jmp .speak

.silent:
    mov rax, 1
    mov rdi, 1
    mov rsi, out_silent
    mov rdx, 2
    syscall
    jmp .exit

.speak:
    mov rax, 1
    mov rdi, 1
    mov rsi, out_speak
    mov rdx, 2
    syscall

.exit:
    mov rax, 60
    xor rdi, rdi
    syscall
