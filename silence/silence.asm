; Chaos2 — silence module
; Input (stdin): "intent|strategy|intensity"
; Output: "1" = stay silent, "0" = speak
;
; Silent when:
;   intent=withdrawal AND strategy=observe
;   intent=philosophical AND intensity < 0.4 (starts with "0.")

section .data
    out_silent   db "1", 10
    out_speak    db "0", 10
    str_withdraw db "withdrawal"
    str_philos   db "philosophical"
    buf          times 64 db 0

section .text
    global _start

; compare buf[0..len-1] with str, return ZF=1 if equal
; rsi = str addr, rcx = len
cmp_intent:
    push rdi
    mov  rdi, buf
    repe cmpsb
    pop  rdi
    ret

_start:
    ; read stdin
    mov rax, 0
    mov rdi, 0
    mov rsi, buf
    mov rdx, 63
    syscall

    ; find first '|', store index in r8
    xor  r8, r8
.find1:
    movzx rax, byte [buf + r8]
    cmp  rax, 0
    je   .speak
    cmp  rax, '|'
    je   .got1
    inc  r8
    jmp  .find1
.got1:
    ; r8 = length of intent string

    ; check intent == "withdrawal" (len=10)
    cmp  r8, 10
    jne  .not_withdrawal
    mov  rsi, str_withdraw
    mov  rcx, 10
    call cmp_intent
    jne  .not_withdrawal

    ; it is withdrawal — silent only if strategy starts with 'o' (observe)
    movzx rax, byte [buf + r8 + 1]
    cmp  rax, 'o'
    je   .silent
    jmp  .speak

.not_withdrawal:
    ; check intent == "philosophical" (len=13)
    cmp  r8, 13
    jne  .speak
    mov  rsi, str_philos
    mov  rcx, 13
    call cmp_intent
    jne  .speak

    ; it is philosophical — find second '|'
    mov  r9, r8
    inc  r9
.find2:
    movzx rax, byte [buf + r9]
    cmp  rax, 0
    je   .speak
    cmp  rax, '|'
    je   .got2
    inc  r9
    jmp  .find2
.got2:
    ; intensity field starts at buf+r9+1
    ; silent only if intensity < 0.3 → second char after '0.' is '0','1','2'
    inc  r9
    movzx rax, byte [buf + r9]     ; should be '0'
    cmp  rax, '0'
    jne  .speak
    inc  r9
    movzx rax, byte [buf + r9]     ; should be '.'
    cmp  rax, '.'
    jne  .speak
    inc  r9
    movzx rax, byte [buf + r9]     ; tenths digit
    cmp  rax, '0'
    je   .silent
    cmp  rax, '1'
    je   .silent
    cmp  rax, '2'
    je   .silent
    jmp  .speak

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
