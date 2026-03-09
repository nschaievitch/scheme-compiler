use std::{
    io::{self, Read},
    u64,
};

use eyre::{Result, eyre};
use num_enum::TryFromPrimitive;

#[allow(non_camel_case_types)]
#[derive(Debug, Clone, Copy, PartialEq, Eq, TryFromPrimitive)]
#[repr(u64)]
pub enum Op {
    LOAD64,
    LOAD_BOOL,
    LOAD_CHAR,
    RETURN,
    ADD1,
    SUB1,
    INTEGER_TO_CHAR,
    CHAR_TO_INTEGER,
    NULLP,
    ZEROP,
    NOT,
    INTEGERP,
    BOOLEANP,
    PLUS,
    TIMES,
    MINUS,
    LESS_THAN,
    EQUALP,
    LOAD_VAR,
    POP_JUMP_IF_FALSE,
    JUMP_FORWARD,
    CONS,
    CAR,
    CDR,
    VECTOR,
    VECTOR_REF,
    VECTOR_SET,
    VECTOR_APPEND,
    STRING,
    STRING_REF,
    STRING_SET,
    STRING_APPEND,
    SETUP_STACK_BASE,
    LABELCALL,
    CLOSURE,
    FUNCALL,
    LOAD_CLOSURE_VAR,
    SLIDE,
    TAILCALL,
    LOAD_VAR_DEFER,
    SET_DEFERRED,
    PUSH_NIL,
}

const HEAP_SIZE: usize = 1 << 10;
const DEBUG_MAX_STACK_SIZE: usize = 1 << 8;

const FIXNUM_MASK: u64 = 0b11;
const FIXNUM_TAG: u64 = 0b00;
const FIXNUM_SHIFT: u64 = 2;

const CHAR_MASK: u64 = 0b11111111;
const CHAR_TAG: u64 = 0b00001111;
const CHAR_SHIFT: u64 = 8;

const BOOLEAN_MASK: u64 = 0b1111111;
const BOOLEAN_TAG: u64 = 0b0011111;
const BOOLEAN_SHIFT: u64 = 7;

const POINTER_MASK: u64 = 0b111;
const POINTER_SHIFT: u64 = 3;
const PAIR_TAG: u64 = 0b001;
const VECTOR_TAG: u64 = 0b010;
const STRING_TAG: u64 = 0b011;
const CLOSURE_TAG: u64 = 0b110;
const FORMAL_TAG: u64 = 0b101;

const NIL_VAL: u64 = 0b111; 

fn read_op(opcode: u64) -> Result<Op> {
    if opcode == 0 {
        return Err(eyre!("Invalid opcode 0"));
    }

    Ok(Op::try_from(opcode - 1)?)
}

macro_rules! define_tag_checker {
    ($name:ident, $mask:expr, $tag:expr) => {
        fn $name(boxed: u64) -> bool {
            (boxed & ($mask as u64)) == ($tag as u64)
        }
    };
}

define_tag_checker!(check_boxed_number, FIXNUM_MASK, FIXNUM_TAG);
define_tag_checker!(check_boxed_character, CHAR_MASK, CHAR_TAG);
define_tag_checker!(check_boxed_boolean, BOOLEAN_MASK, BOOLEAN_TAG);
define_tag_checker!(check_boxed_pair_pointer, POINTER_MASK, PAIR_TAG);
define_tag_checker!(check_boxed_vector_pointer, POINTER_MASK, VECTOR_TAG);
define_tag_checker!(check_boxed_string_pointer, POINTER_MASK, STRING_TAG);
define_tag_checker!(check_boxed_closure_pointer, POINTER_MASK, CLOSURE_TAG);
define_tag_checker!(check_boxed_formal, POINTER_MASK, FORMAL_TAG);

fn read_boxed_number(boxed_num: u64) -> Result<u64> {
    Ok(boxed_num >> FIXNUM_SHIFT)
}

fn box_number(number: u64) -> Result<u64> {
    if number >= 1 << 62 {
        return Err(eyre!("Integer overflow: {}", number));
    } else {
        return Ok(number << FIXNUM_SHIFT);
    }
}

fn read_boxed_character(boxed_char: u64) -> Result<u8> {
    let char_value: u32 = (boxed_char >> CHAR_SHIFT).try_into().or(Err(
        eyre! {"Character value too high: {}", boxed_char >> CHAR_SHIFT},
    ))?;

    let res: u8 = char_value.try_into()?;

    Ok(res)
}

fn box_character(c: u8) -> u64 {
    let b = c as u64;
    (b << CHAR_SHIFT) | CHAR_TAG
}

fn read_boxed_boolean(boxed_boolean: u64) -> bool {
    let boolean_val = (boxed_boolean >> BOOLEAN_SHIFT) & 1;

    return boolean_val == 1;
}

fn box_boolean(boolean: bool) -> u64 {
    let u: u64 = if boolean { 1 } else { 0 };
    (u << BOOLEAN_SHIFT) | BOOLEAN_TAG
}

fn read_boxed_pointer(boxed_pointer: u64) -> Result<usize> {
    return Ok(usize::try_from(boxed_pointer)? >> POINTER_SHIFT);
}

fn box_pair(pair_pointer: usize) -> u64 {
    let u: u64 = pair_pointer.try_into().unwrap();
    (u << POINTER_SHIFT) | PAIR_TAG
}

fn box_vector(vector_pointer: usize) -> u64 {
    let u: u64 = vector_pointer.try_into().unwrap();
    (u << POINTER_SHIFT) | VECTOR_TAG
}

fn box_string(string_pointer: usize) -> u64 {
    let u: u64 = string_pointer.try_into().unwrap();
    (u << POINTER_SHIFT) | STRING_TAG
}

fn box_closure(closure_pointer: usize) -> u64 {
    let u: u64 = closure_pointer.try_into().unwrap();
    (u << POINTER_SHIFT) | CLOSURE_TAG
}

fn box_formal(formal_offset: usize) -> u64 {
    let u: u64 = formal_offset.try_into().unwrap();
    (u << POINTER_SHIFT) | FORMAL_TAG
}

fn print_boxed(boxed: u64, heap: &[u64]) -> Result<String> {
    if check_boxed_number(boxed) {
        return Ok(format!("{}", read_boxed_number(boxed)?));
    }

    if check_boxed_character(boxed) {
        return Ok(format!("#\\{}", char::from(read_boxed_character(boxed)?)));
    }

    if check_boxed_boolean(boxed) {
        let b = read_boxed_boolean(boxed);
        if b {
            return Ok("#t".to_string());
        } else {
            return Ok("#f".to_string());
        }
    }

    if check_boxed_pair_pointer(boxed) {
        let p = read_boxed_pointer(boxed)?;
        let s1 = print_boxed(heap[p], heap)?;
        let s2 = print_boxed(heap[p + 1], heap)?;

        return Ok(format!("({} . {})", s1, s2));
    }

    if check_boxed_vector_pointer(boxed) {
        let p = read_boxed_pointer(boxed)?;

        let len: usize = heap[p].try_into()?;

        let mut s = String::new();
        s.push_str("#(");

        for i in 1..len {
            s.push_str(&print_boxed(heap[p + i], heap)?);
            s.push(' ');
        }

        s.push_str(&print_boxed(heap[p + len], heap)?);
        s.push(')');

        return Ok(s);
    }

    if check_boxed_string_pointer(boxed) {
        let p = read_boxed_pointer(boxed)?;

        let len: usize = heap[p].try_into()?;

        let str_word_len = len / 8 + 1;

        let mut s = String::new();

        s.push('"');

        for i in 1..=str_word_len {
            let word = heap[p + i];
            let bytes = word.to_le_bytes();
            bytes.iter().for_each(|b| {
                if *b != 0 {
                    s.push(char::from(b.clone()));
                }
            })
        }

        s.push('"');

        return Ok(s);
    }

    if check_boxed_closure_pointer(boxed) {
        let p = read_boxed_pointer(boxed)?;
        return Ok(format!("<closure over label {}>", p));
    }

    if boxed == NIL_VAL {
        return Ok("'()".to_owned());
    }

    Err(eyre!("Invalid tagged value {}", boxed))
}

fn readword(code: &[u64], pc: &mut usize) -> Result<u64> {
    if *pc >= code.len() {
        return Err(eyre!("Program ends without return"));
    }

    let word = code[*pc];
    *pc += 1;
    Ok(word)
}

fn interpret(code: Vec<u64>, heap: &mut Box<[u64; HEAP_SIZE]>) -> Result<u64> {
    let mut pc: usize = 0;
    let mut stack: Vec<u64> = Vec::new();
    let mut control_stack: Vec<usize> = Vec::new();
    // let mut heap: Box<[u64; HEAP_SIZE]> = Box::new([0; HEAP_SIZE]);
    let mut heap_pointer: usize = 0;
    let mut stack_base_pointer: usize = 0;

    let mut current_closure: usize = 0;

    // first word should point to main function
    let main_pos: usize = readword(&code, &mut pc)?.try_into()?;
    pc = main_pos;

    loop {
        let instr = read_op(readword(&code, &mut pc)?)?;
        dbg!(&instr, &pc, &code[pc - 1]);

        debug_assert!(stack.len() <= DEBUG_MAX_STACK_SIZE, "stack overflow");

        match instr {
            Op::LOAD64 => {
                let boxed_num = readword(&code, &mut pc)?;
                if !check_boxed_number(boxed_num) {
                    return Err(eyre!("Invalid tagged number value {}", boxed_num));
                }

                stack.push(boxed_num)
            }

            Op::LOAD_CHAR => {
                let boxed_char = readword(&code, &mut pc)?;
                if !check_boxed_character(boxed_char) {
                    return Err(eyre!("Invalid tagged character value {}", boxed_char));
                }

                stack.push(boxed_char)
            }

            Op::LOAD_BOOL => {
                let boxed_boolean = readword(&code, &mut pc)?;
                if !check_boxed_boolean(boxed_boolean) {
                    return Err(eyre!("Invalid tagged boolean value {}", boxed_boolean));
                }

                stack.push(boxed_boolean)
            }

            Op::RETURN => {
                dbg!(&control_stack);

                if control_stack.len() > 0
                    && check_boxed_closure_pointer(
                        control_stack.last().unwrap().clone().try_into()?,
                    )
                {
                    control_stack.pop();
                }

                let val = stack.pop().ok_or(eyre!("No return value"))?;

                dbg!(print_boxed(val.clone(), &heap[..])?);

                if stack_base_pointer == 0 {
                    dbg!(&stack);
                    dbg!(&heap[..10]);
                    return Ok(val);
                }

                let new_pc: usize = control_stack
                    .pop()
                    .ok_or(eyre!("Tried to pop pc from empty control stack"))?;

                let caller_closure_boxed: usize = control_stack
                    .pop()
                    .ok_or(eyre!("Tried to pop caller closure from empty control stack"))?;

                if check_boxed_closure_pointer(caller_closure_boxed.try_into()?) {
                    current_closure = read_boxed_pointer(caller_closure_boxed.try_into()?)?;
                }

                let new_bp: usize = control_stack
                    .pop()
                    .ok_or(eyre!("Tried to pop bp from empty control stack"))?;

                stack.truncate(stack_base_pointer);

                pc = new_pc;
                stack_base_pointer = new_bp;

                // dbg!(&code, &pc);

                stack.push(val);
            }

            Op::ADD1 => {
                let val = stack
                    .pop()
                    .ok_or(eyre!("Unexpected call to add1 without arguments"))?;

                if !check_boxed_number(val) {
                    return Err(eyre!(
                        "Called add1 on a value that is not a number: {}",
                        print_boxed(val, &heap[..])?
                    ));
                }

                let new_val = box_number(read_boxed_number(val)? + 1)?;
                stack.push(new_val);
            }

            Op::SUB1 => {
                let val = stack
                    .pop()
                    .ok_or(eyre!("Unexpected call to sub1 without arguments"))?;

                if !check_boxed_number(val) {
                    return Err(eyre!(
                        "Called sub1 on a value that is not a number: {}",
                        print_boxed(val, &heap[..])?
                    ));
                }

                let new_val = box_number(read_boxed_number(val)? - 1)?;
                stack.push(new_val);
            }

            Op::BOOLEANP => {
                let val = stack
                    .pop()
                    .ok_or(eyre!("Unexpected call to boolean? without arguments"))?;

                if check_boxed_boolean(val) {
                    stack.push(box_boolean(true));
                } else {
                    stack.push(box_boolean(false));
                }
            }

            Op::INTEGERP => {
                let val = stack
                    .pop()
                    .ok_or(eyre!("Unexpected call to integer? without arguments"))?;

                if check_boxed_number(val) {
                    stack.push(box_boolean(true));
                } else {
                    stack.push(box_boolean(false));
                }
            }

            Op::NULLP => {
                let val = stack
                    .pop()
                    .ok_or(eyre!("Unexpected call to null? without arguments"))?;

                if val == NIL_VAL {
                    stack.push(box_boolean(true));
                } else {
                    stack.push(box_boolean(false));
                }
            }

            Op::ZEROP => {
                let val = stack
                    .pop()
                    .ok_or(eyre!("Unexpected call to zero? without arguments"))?;

                if !check_boxed_number(val) {
                    return Err(eyre!(
                        "Called zero? on a value that is not a number: {}",
                        print_boxed(val, &heap[..])?
                    ));
                }

                if val == box_number(0)? {
                    stack.push(box_boolean(true));
                } else {
                    stack.push(box_boolean(false));
                }
            }

            Op::INTEGER_TO_CHAR => {
                let val = stack
                    .pop()
                    .ok_or(eyre!("Unexpected call to integer->char without arguments"))?;

                if !check_boxed_number(val) {
                    return Err(eyre!(
                        "Called integer->char on a value that is not a number: {}",
                        print_boxed(val, &heap[..])?
                    ));
                }

                let num: u8 = read_boxed_number(val)?.try_into()?;

                stack.push(box_character(num));
            }

            Op::CHAR_TO_INTEGER => {
                let val = stack
                    .pop()
                    .ok_or(eyre!("Unexpected call to char->integer without arguments"))?;

                if !check_boxed_character(val) {
                    return Err(eyre!(
                        "Called char->integer on a value that is not a character: {}",
                        print_boxed(val, &heap[..])?
                    ));
                }

                let char = read_boxed_character(val)?;

                let b = char as u64;

                stack.push(box_number(b)?);
            }

            Op::NOT => {
                let val = stack
                    .pop()
                    .ok_or(eyre!("Unexpected call to not without arguments"))?;

                if !check_boxed_boolean(val) {
                    return Err(eyre!(
                        "Called not on a value that is not a boolean: {}",
                        print_boxed(val, &heap[..])?
                    ));
                }

                let bool = read_boxed_boolean(val);

                stack.push(box_boolean(!bool));
            }

            Op::PLUS => {
                let val1 = stack
                    .pop()
                    .ok_or(eyre!("Unexpected call to + without arguments"))?;

                let val2 = stack
                    .pop()
                    .ok_or(eyre!("Unexpected call to + with one argument"))?;

                if !check_boxed_number(val1) {
                    return Err(eyre!(
                        "Non number argument for + {}",
                        print_boxed(val1, &heap[..])?
                    ));
                }

                if !check_boxed_number(val2) {
                    return Err(eyre!(
                        "Non number argument for + {}",
                        print_boxed(val2, &heap[..])?
                    ));
                }

                let num1 = read_boxed_number(val1)?;
                let num2 = read_boxed_number(val2)?;

                stack.push(box_number(num1 + num2)?)
            }

            Op::MINUS => {
                let val1 = stack
                    .pop()
                    .ok_or(eyre!("Unexpected call to - without arguments"))?;

                let val2 = stack
                    .pop()
                    .ok_or(eyre!("Unexpected call to - with one argument"))?;

                if !check_boxed_number(val1) {
                    return Err(eyre!(
                        "Non number argument for - {}",
                        print_boxed(val1, &heap[..])?
                    ));
                }

                if !check_boxed_number(val2) {
                    return Err(eyre!(
                        "Non number argument for - {}",
                        print_boxed(val2, &heap[..])?
                    ));
                }

                let num1 = read_boxed_number(val1)?;
                let num2 = read_boxed_number(val2)?;

                stack.push(box_number(num1 - num2)?)
            }

            Op::TIMES => {
                let val1 = stack
                    .pop()
                    .ok_or(eyre!("Unexpected call to * without arguments"))?;

                let val2 = stack
                    .pop()
                    .ok_or(eyre!("Unexpected call to * with one argument"))?;

                if !check_boxed_number(val1) {
                    return Err(eyre!(
                        "Non number argument for * {}",
                        print_boxed(val1, &heap[..])?
                    ));
                }

                if !check_boxed_number(val2) {
                    return Err(eyre!(
                        "Non number argument for * {}",
                        print_boxed(val2, &heap[..])?
                    ));
                }

                let num1 = read_boxed_number(val1)?;
                let num2 = read_boxed_number(val2)?;

                stack.push(box_number(num1 * num2)?)
            }

            Op::LESS_THAN => {
                let val1 = stack
                    .pop()
                    .ok_or(eyre!("Unexpected call to < without arguments"))?;

                let val2 = stack
                    .pop()
                    .ok_or(eyre!("Unexpected call to < with one argument"))?;

                if !check_boxed_number(val1) {
                    return Err(eyre!(
                        "Non number argument for < {}",
                        print_boxed(val1, &heap[..])?
                    ));
                }

                if !check_boxed_number(val2) {
                    return Err(eyre!(
                        "Non number argument for < {}",
                        print_boxed(val2, &heap[..])?
                    ));
                }

                let num1 = read_boxed_number(val1)?;
                let num2 = read_boxed_number(val2)?;

                stack.push(box_boolean(num1 < num2))
            }

            Op::EQUALP => {
                let val1 = stack
                    .pop()
                    .ok_or(eyre!("Unexpected call to = without arguments"))?;

                let val2 = stack
                    .pop()
                    .ok_or(eyre!("Unexpected call to = with one argument"))?;

                stack.push(box_boolean(val1 == val2))
            }

            Op::LOAD_VAR => {
                let stack_index: usize = readword(&code, &mut pc)?.try_into()?;
                dbg!(&stack, &stack_index);

                let val: u64 = *(stack.get(stack_index + stack_base_pointer)
                    .ok_or(eyre!("Invalid variable load from bad position, tried to grab position {} from stack of length {}", stack_index + stack_base_pointer, stack.len()))?);

                stack.push(val.clone())
            }

            Op::POP_JUMP_IF_FALSE => {
                let val = stack.pop().ok_or(eyre!(
                    "Unexpected call to POP_JUMP_IF_FALSE without arguments"
                ))?;

                if check_boxed_boolean(val) {
                    let condition = read_boxed_boolean(val);

                    if !condition {
                        let offset: usize = readword(&code, &mut pc)?.try_into()?;

                        pc += offset;
                    } else {
                        pc += 1;
                    }
                } else {
                    // if not #f, it's truthy
                    pc += 1;
                }
            }

            Op::JUMP_FORWARD => {
                let offset: usize = readword(&code, &mut pc)?.try_into()?;

                pc += offset;
            }

            Op::CONS => {
                dbg!(&stack);
                let val1 = stack
                    .pop()
                    .ok_or(eyre!("Unexpected call to cons without arguments"))?;

                let val2 = stack
                    .pop()
                    .ok_or(eyre!("Unexpected call to cons with one argument"))?;

                stack.push(box_pair(heap_pointer));

                heap[heap_pointer] = val1;
                heap[heap_pointer + 1] = val2;

                heap_pointer += 2;
            }

            Op::CAR => {
                let val1 = stack
                    .pop()
                    .ok_or(eyre!("Unexpected call to car without arguments"))?;

                if !check_boxed_pair_pointer(val1) {
                    return Err(eyre!(
                        "Non pair argument for car {}",
                        print_boxed(val1, &heap[..])?
                    ));
                }

                let pair_pointer = read_boxed_pointer(val1)?;

                stack.push(heap[pair_pointer]);
            }

            Op::CDR => {
                let val1 = stack
                    .pop()
                    .ok_or(eyre!("Unexpected call to cdr without arguments"))?;

                if !check_boxed_pair_pointer(val1) {
                    return Err(eyre!(
                        "Non pair argument for cdr {}",
                        print_boxed(val1, &heap[..])?
                    ));
                }

                let pair_pointer = read_boxed_pointer(val1)?;

                stack.push(heap[pair_pointer + 1]);
            }

            Op::VECTOR => {
                let vec_len: usize = readword(&code, &mut pc)?.try_into()?;

                if stack.len() < vec_len {
                    return Err(eyre!(
                        "Invalid stack state; not enough values for vector length"
                    ));
                }

                heap[heap_pointer] = vec_len.try_into()?;

                for i in 1..=vec_len {
                    heap[heap_pointer + i] = stack
                        .pop()
                        .ok_or(eyre!("Error popping from stack to create vector"))?;
                }

                stack.push(box_vector(heap_pointer));
                heap_pointer += vec_len + 1;
            }

            Op::VECTOR_REF => {
                let vec_pointer = stack.pop().ok_or(eyre!("Invalid call to vector_ref"))?;

                let vec_position_boxed = stack.pop().ok_or(eyre!("Invalid call to vector_ref"))?;

                if !check_boxed_vector_pointer(vec_pointer) {
                    return Err(eyre!(
                        "Invalid vector pointer {}",
                        print_boxed(vec_pointer, &heap[..])?
                    ));
                }

                if !check_boxed_number(vec_position_boxed) {
                    return Err(eyre!(
                        "Invalid vector position, is not a number {}",
                        print_boxed(vec_position_boxed, &heap[..])?
                    ));
                }

                let pointer = read_boxed_pointer(vec_pointer)?;
                let vec_len = heap[pointer];

                let vec_position = read_boxed_number(vec_position_boxed)?;

                if vec_position >= vec_len {
                    return Err(eyre!(
                        "Attempted to read out of vector bounds: position {vec_position} in vector of length {vec_len}"
                    ));
                }

                let mut pos: usize = vec_position.try_into()?;
                pos += pointer + 1;

                stack.push(heap[pos]);
            }

            Op::VECTOR_SET => {
                let vec_pointer = stack.pop().ok_or(eyre!("Invalid call to vector_set"))?;

                let vec_position_boxed = stack.pop().ok_or(eyre!("Invalid call to vector_set"))?;

                let vec_value = stack.pop().ok_or(eyre!("Invalid call to vector_set"))?;

                if !check_boxed_vector_pointer(vec_pointer) {
                    return Err(eyre!(
                        "Invalid vector pointer {}",
                        print_boxed(vec_pointer, &heap[..])?
                    ));
                }

                if !check_boxed_number(vec_position_boxed) {
                    return Err(eyre!(
                        "Invalid vector position, is not a number {}",
                        print_boxed(vec_position_boxed, &heap[..])?
                    ));
                }

                let pointer = read_boxed_pointer(vec_pointer)?;
                let vec_len = heap[pointer];

                let vec_position = read_boxed_number(vec_position_boxed)?;

                if vec_position >= vec_len {
                    return Err(eyre!(
                        "Attempted to read out of vector bounds: position {vec_position} in vector of length {vec_len}"
                    ));
                }

                let mut pos: usize = vec_position.try_into()?;
                pos += pointer + 1;

                heap[pos] = vec_value;

                stack.push(vec_pointer);
            }

            Op::VECTOR_APPEND => {
                let vec_boxed_1 = stack.pop().ok_or(eyre!("Invalid call to vector-append"))?;
                let vec_boxed_2 = stack.pop().ok_or(eyre!("Invalid call to vector-append"))?;

                if !check_boxed_vector_pointer(vec_boxed_1)
                    || !check_boxed_vector_pointer(vec_boxed_2)
                {
                    return Err(eyre!("Invalid argument to vector-append, not a vector"));
                }

                let vec_pointer_1 = read_boxed_pointer(vec_boxed_1)?;
                let vec_pointer_2 = read_boxed_pointer(vec_boxed_2)?;

                let len_vec_1: usize = heap[vec_pointer_1].try_into()?;
                let len_vec_2: usize = heap[vec_pointer_2].try_into()?;

                let new_len = len_vec_1 + len_vec_2;

                heap[heap_pointer] = new_len.try_into()?;

                for i in 1..=len_vec_1 {
                    heap[heap_pointer + i] = heap[vec_pointer_1 + i]
                }

                for i in 1..=len_vec_2 {
                    heap[heap_pointer + len_vec_1 + i] = heap[vec_pointer_2 + i]
                }

                stack.push(box_vector(heap_pointer));

                heap_pointer += len_vec_1 + len_vec_2 + 1;
            }

            Op::STRING => {
                let string_len: usize = readword(&code, &mut pc)?.try_into()?;

                if stack.len() < string_len {
                    return Err(eyre!(
                        "Invalid stack state; not enough values for string length"
                    ));
                }

                heap[heap_pointer] = string_len.try_into()?;

                let in_memory_size = (string_len / 8) + 1;

                let mut bytes: Vec<u8> = Vec::with_capacity(string_len);

                for _ in 1..=string_len {
                    let boxed_char = stack
                        .pop()
                        .ok_or(eyre!("Error popping from stack to create string"))?;

                    if !check_boxed_character(boxed_char) {
                        return Err(eyre!(
                            "Invalid non-char element in string call {}",
                            print_boxed(boxed_char, &heap[..])?
                        ));
                    }

                    let char = read_boxed_character(boxed_char)?;
                    bytes.push(char);
                }

                for (i, chunk) in bytes.chunks(8).enumerate() {
                    let mut buf = [0u8; 8];
                    buf[..chunk.len()].copy_from_slice(chunk);

                    let w = u64::from_le_bytes(buf);
                    heap[heap_pointer + i + 1] = w;
                }

                stack.push(box_string(heap_pointer));
                heap_pointer += in_memory_size + 1;
            }

            Op::STRING_REF => {
                let string_pointer = stack.pop().ok_or(eyre!("Invalid call to string_ref"))?;

                let string_position_boxed =
                    stack.pop().ok_or(eyre!("Invalid call to string_ref"))?;

                if !check_boxed_string_pointer(string_pointer) {
                    return Err(eyre!(
                        "Invalid string pointer {}",
                        print_boxed(string_pointer, &heap[..])?
                    ));
                }

                if !check_boxed_number(string_position_boxed) {
                    return Err(eyre!(
                        "Invalid string position, is not a number {}",
                        print_boxed(string_position_boxed, &heap[..])?
                    ));
                }

                let pointer = read_boxed_pointer(string_pointer)?;
                dbg!(&pointer);
                let string_len = heap[pointer];

                let string_position = read_boxed_number(string_position_boxed)?;

                if string_position >= string_len {
                    return Err(eyre!(
                        "Attempted to read out of string bounds: position {string_position} in string of length {string_len}"
                    ));
                }

                let string_position_u: usize = string_position.try_into()?;
                let pos_word: usize = string_position_u / 8;

                let word = heap[pointer + pos_word + 1];

                let word_bytes = word.to_le_bytes();

                let char = word_bytes[string_position_u % 8];

                stack.push(box_character(char));
            }

            Op::STRING_SET => {
                let string_pointer = stack.pop().ok_or(eyre!("Invalid call to string_ref"))?;

                let string_position_boxed =
                    stack.pop().ok_or(eyre!("Invalid call to string_ref"))?;

                let new_char_boxed = stack.pop().ok_or(eyre!("Invalid call to string_ref"))?;

                if !check_boxed_string_pointer(string_pointer) {
                    return Err(eyre!(
                        "Invalid string pointer {}",
                        print_boxed(string_pointer, &heap[..])?
                    ));
                }

                if !check_boxed_number(string_position_boxed) {
                    return Err(eyre!(
                        "Invalid string position, is not a number {}",
                        print_boxed(string_position_boxed, &heap[..])?
                    ));
                }

                if !check_boxed_character(new_char_boxed) {
                    return Err(eyre!(
                        "Invalid string modification, is not a character {}",
                        print_boxed(string_position_boxed, &heap[..])?
                    ));
                }

                let pointer = read_boxed_pointer(string_pointer)?;
                let string_len = heap[pointer];

                let string_position = read_boxed_number(string_position_boxed)?;

                let new_char = read_boxed_character(new_char_boxed)?;

                dbg!(&heap[..4], &string_position, &new_char, &pointer);
                if string_position >= string_len {
                    return Err(eyre!(
                        "Attempted to read out of string bounds: position {string_position} in string of length {string_len}"
                    ));
                }

                let string_position_u: usize = string_position.try_into()?;
                let pos_word: usize = string_position_u / 8;

                let word = heap[pointer + pos_word + 1];

                let mut word_bytes = word.to_le_bytes();

                word_bytes[string_position_u % 8] = new_char;

                heap[pointer + pos_word + 1] = u64::from_le_bytes(word_bytes);

                stack.push(string_pointer);
                dbg!(&heap[..4]);
            }

            Op::STRING_APPEND => {
                let string_1_pointer_boxed =
                    stack.pop().ok_or(eyre!("Invalid call to string_append"))?;
                let string_2_pointer_boxed =
                    stack.pop().ok_or(eyre!("Invalid call to string_append"))?;

                if !check_boxed_string_pointer(string_1_pointer_boxed) {
                    return Err(eyre!(
                        "Invalid string pointer {}",
                        print_boxed(string_1_pointer_boxed, &heap[..])?
                    ));
                }

                if !check_boxed_string_pointer(string_2_pointer_boxed) {
                    return Err(eyre!(
                        "Invalid string pointer {}",
                        print_boxed(string_2_pointer_boxed, &heap[..])?
                    ));
                }

                let string_1 = read_boxed_pointer(string_1_pointer_boxed)?;
                let string_2 = read_boxed_pointer(string_2_pointer_boxed)?;

                let str_len_1: usize = heap[string_1].try_into()?;
                let str_len_2: usize = heap[string_2].try_into()?;

                let new_str_len = str_len_1 + str_len_2;

                let mut str_bytes_1: Vec<u8> = Vec::new();
                let mut str_bytes_2: Vec<u8> = Vec::new();

                let str_1_word_len = str_len_1 / 8 + 1;
                let str_2_word_len = str_len_2 / 8 + 1;

                for i in 1..=str_1_word_len {
                    let word = heap[string_1 + i].to_le_bytes();

                    str_bytes_1.extend_from_slice(&word);
                }

                for i in 1..=str_2_word_len {
                    let word = heap[string_2 + i].to_le_bytes();

                    str_bytes_2.extend_from_slice(&word);
                }

                let mut str_bytes: Vec<u8> = Vec::new();

                str_bytes.extend_from_slice(&str_bytes_1[..str_len_1]);
                str_bytes.extend_from_slice(&str_bytes_2[..str_len_2]);

                stack.push(box_string(heap_pointer));

                heap[heap_pointer] = new_str_len.try_into()?;
                heap_pointer += 1;

                for (i, chunk) in str_bytes.chunks(8).enumerate() {
                    let mut buf = [0u8; 8];
                    buf[..chunk.len()].copy_from_slice(chunk);

                    let w = u64::from_le_bytes(buf);
                    heap[heap_pointer + i] = w;
                }

                heap_pointer += new_str_len / 8 + 1;
            }

            Op::SETUP_STACK_BASE => {
                control_stack.push(stack_base_pointer);

                control_stack.push(stack.len());
            }

            Op::LABELCALL => {
                stack_base_pointer = control_stack
                    .pop()
                    .ok_or(eyre!("labelcall with empty control stack"))?;
                control_stack.push(0); // no caller closure to restore
                control_stack.push(pc + 1);

                let label_add: usize = readword(&code, &mut pc)?.try_into()?;
                let jump_pos: usize = code[label_add].try_into()?;

                pc = jump_pos;
            }

            Op::CLOSURE => {
                let label_add = readword(&code, &mut pc)?;

                heap[heap_pointer] = label_add;

                let args_len: usize = readword(&code, &mut pc)?.try_into()?;

                if stack.len() < args_len {
                    return Err(eyre!(
                        "Invalid stack state; not enough values for closure length"
                    ));
                }

                for i in 1..=args_len {
                    let val = stack
                        .pop()
                        .ok_or(eyre!("Error popping from stack to create closure"))?;

                    if check_boxed_formal(val) {
                        control_stack.push(heap_pointer + i);
                    }

                    heap[heap_pointer + i] = val;
                }

                stack.push(box_closure(heap_pointer));
                heap_pointer += args_len + 1;
            }

            Op::FUNCALL => {
                stack_base_pointer = control_stack
                    .pop()
                    .ok_or(eyre!("funcall with empty control stack"))?;
                control_stack.push(box_closure(current_closure).try_into()?);
                control_stack.push(pc);

                let boxed_closure_pointer = stack
                    .pop()
                    .ok_or(eyre!("Tried to funcall with no closure on stack"))?;

                if !check_boxed_closure_pointer(boxed_closure_pointer) {
                    dbg!(&stack);
                    return Err(eyre!("Invalid closure pointer {}", boxed_closure_pointer));
                }

                control_stack.push(boxed_closure_pointer.try_into()?);
                current_closure = read_boxed_pointer(boxed_closure_pointer)?;

                let label_add: usize = heap[current_closure].try_into()?;
                let jump_pos: usize = code[label_add].try_into()?;

                dbg!(&stack, &jump_pos, &label_add);
                pc = jump_pos;
            }

            Op::LOAD_CLOSURE_VAR => {
                let closure_pointer = current_closure;

                let var_index: usize = readword(&code, &mut pc)?.try_into()?;

                let val = heap[closure_pointer + var_index + 1];

                stack.push(val)
            }

            Op::SLIDE => {
                let n: usize = readword(&code, &mut pc)?.try_into()?;
                let val = stack
                    .pop()
                    .ok_or(eyre!("Tried to slide from empty stack"))?;

                stack.truncate(stack.len() - n);
                stack.push(val);
            }

            Op::TAILCALL => {
                // Discard the 2 values SETUP_STACK_BASE pushed onto control_stack
                control_stack.pop();
                control_stack.pop();

                let n_args: usize = readword(&code, &mut pc)?.try_into()?;
                let start = stack.len() - (n_args + 1);

                stack.copy_within(start.., stack_base_pointer);

                stack.truncate(stack_base_pointer + n_args + 1);

                let boxed_closure_pointer = stack
                    .pop()
                    .ok_or(eyre!("Tried to funcall with no closure on stack"))?;

                if !check_boxed_closure_pointer(boxed_closure_pointer) {
                    dbg!(&stack);
                    return Err(eyre!("Invalid closure pointer {}", boxed_closure_pointer));
                }

                current_closure = read_boxed_pointer(boxed_closure_pointer)?;

                let label_add: usize = heap[current_closure].try_into()?;
                let jump_pos: usize = code[label_add].try_into()?;

                dbg!(&stack, &jump_pos, &label_add);
                pc = jump_pos;
            }

            Op::LOAD_VAR_DEFER => {
                let var_offset: usize = readword(&code, &mut pc)?.try_into()?;
                let boxed = box_formal(var_offset);
                stack.push(boxed);
            }

            Op::SET_DEFERRED => {
                let n = readword(&code, &mut pc)?;

                dbg!(&n, &control_stack, &heap[..10]);
                for _i in 0..n {
                    let idx = control_stack
                        .pop()
                        .ok_or(eyre!("Tried to pop from empty control stack"))?;
                    let offset = read_boxed_pointer(heap[idx])?;

                    heap[idx] = *(stack.get(offset + stack_base_pointer)
                    .ok_or(eyre!("Invalid variable load from bad position, tried to grab position {} from stack of length {}", idx + stack_base_pointer, stack.len()))?);
                }
                dbg!(&heap[..10]);
            }

            Op::PUSH_NIL => {
                stack.push(NIL_VAL);
            }
        }
    }
}

fn read_stdin() -> Result<Vec<u64>> {
    let stdin = io::stdin();
    let mut handle = stdin.lock();
    let mut buffer = [0u8; 8];

    let mut code: Vec<u64> = Vec::new();

    while handle.read_exact(&mut buffer).is_ok() {
        let val = u64::from_le_bytes(buffer);
        code.push(val);
    }

    Ok(code)
}

fn main() -> Result<()> {
    let mut heap: Box<[u64; HEAP_SIZE]> = Box::new([0; HEAP_SIZE]);
    let code = read_stdin()?;

    let ret = interpret(code, &mut heap)?;

    // dbg!(ret);
    let print_val = print_boxed(ret, &heap[..])?;

    println!("{}", print_val);

    Ok(())
}
