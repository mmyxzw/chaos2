; Chaos2 — silence module
; Assembly x86-64: decides when Chaos should NOT respond
; Input (stdin): "intent|strategy|intensity"
; Output: "1" = stay silent, "0" = speak
;
; Silent when:
;   intent=withdrawal AND strategy=observe
;   intent=philosophical AND intensity starts with '0'

section .data
    out_silent  db "1", 10
    out_speak   db "0", 10
    buf         times 64 db 0

section .text
    global _start

_start:
    ; read stdin
    mov rax, 0
    mov rdi, 0
    mov rsi, buf
    mov rdx, 63
    syscall

    ; find first '|' → split intent / strategy
    mov rcx, 0
.find_sep1:
    movzx rax, byte [buf + rcx]
    cmp rax, 0
    je .speak
    cmp rax, '|'
    je .got_sep1
    inc rcx
    jmp .find_sep1

.got_sep1:
    ; rcx = index of first '|'
    ; buf[0] = first char of intent
    movzx rbx, byte [buf]

    ; check intent=withdrawal ('w')
    cmp rbx, 'w'
    jne .check_philosophical

    ; intent is withdrawal — silent only if strategy starts with 'o' (observe)
    movzx rax, byte [buf + rcx + 1]
    cmp rax, 'o'
    je .silent
    jmp .speak

.check_philosophical:
    ; check intent=philosophical ('p')
    cmp rbx, 'p'
    jne .speak

    ; find second '|' → intensity field
    mov rdx, rcx
    inc rdx
.find_sep2:
    movzx rax, byte [buf + rdx]
    cmp rax, 0
    je .speak
    cmp rax, '|'
    je .got_sep2
    inc rdx
    jmp .find_sep2

.got_sep2:
    inc rdx
    movzx rax, byte [buf + rdx]
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
