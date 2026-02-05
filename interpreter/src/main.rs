use std::{
    io::{self, Read},
    u64,
};

use eyre::{Result, eyre};
use num_enum::TryFromPrimitive;

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
}

const HEAP_SIZE: usize = 2 << 10;

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

fn read_boxed_character(boxed_char: u64) -> Result<char> {
    let char_value: u32 = (boxed_char >> CHAR_SHIFT).try_into().or(Err(
        eyre! {"Character value too high: {}", boxed_char >> CHAR_SHIFT},
    ))?;

    char::from_u32(char_value).ok_or(eyre!("Invalid character value {}", char_value))
}

fn box_character(c: char) -> u64 {
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

fn print_boxed(boxed: u64) -> Result<String> {
    if check_boxed_number(boxed) {
        return Ok(format!("{}", read_boxed_number(boxed)?));
    }

    if check_boxed_character(boxed) {
        return Ok(format!("#\\{}", read_boxed_character(boxed)?));
    }

    if check_boxed_boolean(boxed) {
        let b = read_boxed_boolean(boxed);
        if b {
            return Ok("#t".to_string());
        } else {
            return Ok("#f".to_string());
        }
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

fn interpret(code: Vec<u64>) -> Result<u64> {
    let mut pc: usize = 0;
    let mut stack: Vec<u64> = Vec::new();
    let mut heap: Box<[u64; HEAP_SIZE]> = Box::new([0; HEAP_SIZE]);
    let mut heap_pointer: usize = 0;

    loop {
        let instr = read_op(readword(&code, &mut pc)?)?;
        // dbg!(instr, pc);
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
                let val = stack.pop().ok_or(eyre!("No return value"));
                return val;
            }

            Op::ADD1 => {
                let val = stack
                    .pop()
                    .ok_or(eyre!("Unexpected call to add1 without arguments"))?;

                if !check_boxed_number(val) {
                    return Err(eyre!(
                        "Called add1 on a value that is not a number: {}",
                        print_boxed(val)?
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
                        print_boxed(val)?
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

                if val == 0 {
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
                        print_boxed(val)?
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
                        print_boxed(val)?
                    ));
                }

                let num: u32 = read_boxed_number(val)?.try_into()?;

                let c: char =
                    char::from_u32(num).ok_or(eyre!("Invalid value for a character {}", num))?;

                stack.push(box_character(c));
            }

            Op::CHAR_TO_INTEGER => {
                let val = stack
                    .pop()
                    .ok_or(eyre!("Unexpected call to char->integer without arguments"))?;

                if !check_boxed_character(val) {
                    return Err(eyre!(
                        "Called char->integer on a value that is not a character: {}",
                        print_boxed(val)?
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
                        print_boxed(val)?
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
                    return Err(eyre!("Non number argument for + {}", print_boxed(val1)?));
                }

                if !check_boxed_number(val2) {
                    return Err(eyre!("Non number argument for + {}", print_boxed(val2)?));
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
                    return Err(eyre!("Non number argument for - {}", print_boxed(val1)?));
                }

                if !check_boxed_number(val2) {
                    return Err(eyre!("Non number argument for - {}", print_boxed(val2)?));
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
                    return Err(eyre!("Non number argument for * {}", print_boxed(val1)?));
                }

                if !check_boxed_number(val2) {
                    return Err(eyre!("Non number argument for * {}", print_boxed(val2)?));
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
                    return Err(eyre!("Non number argument for < {}", print_boxed(val1)?));
                }

                if !check_boxed_number(val2) {
                    return Err(eyre!("Non number argument for < {}", print_boxed(val2)?));
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

                let val: u64 = *(stack.get(stack_index)
                    .ok_or(eyre!("Invalid variable load from bad position, tried to grab position {} from stack of length {}", stack_index, stack.len()))?);

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
                    return Err(eyre!("Non pair argument for car {}", print_boxed(val1)?));
                }

                let pair_pointer = read_boxed_pointer(val1)?;

                stack.push(heap[pair_pointer]);
            }

            Op::CDR => {
                let val1 = stack
                    .pop()
                    .ok_or(eyre!("Unexpected call to cdr without arguments"))?;

                if !check_boxed_pair_pointer(val1) {
                    return Err(eyre!("Non pair argument for cdr {}", print_boxed(val1)?));
                }

                let pair_pointer = read_boxed_pointer(val1)?;

                stack.push(heap[pair_pointer + 1]);
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
    let code = read_stdin()?;

    let ret = interpret(code)?;

    // dbg!(ret);
    let print_val = print_boxed(ret)?;

    println!("{}", print_val);

    Ok(())
}
