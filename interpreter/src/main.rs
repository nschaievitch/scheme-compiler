use std::io::{self, Read};

use eyre::{Result, eyre};
use num_enum::TryFromPrimitive;

#[derive(Debug, Clone, Copy, PartialEq, Eq, TryFromPrimitive)]
#[repr(u64)]
pub enum Op {
    LOAD64,
    LOAD_BOOL,
    LOAD_CHAR,
    RETURN,
}

const FIXNUM_MASK: u64 = 0b11;
const FIXNUM_TAG: u64 = 0b00;
const FIXNUM_SHIFT: u64 = 2;

const CHAR_MASK: u64 = 0b11111111;
const CHAR_TAG: u64 = 0b00001111;
const CHAR_SHIFT: u64 = 8;

const BOOLEAN_MASK: u64 = 0b1111111;
const BOOLEAN_TAG: u64 = 0b0011111;
const BOOLEAN_SHIFT: u64 = 7;

fn read_op(opcode: u64) -> Result<Op> {
    // TODO: improve ugly pattern
    // Python enums start from 1, so we need to decrease by 1
    // throw error if 0
    if opcode == 0 {
        return Err(eyre!("Invalid opcode 0"));
    }
    Ok(Op::try_from(opcode - 1)?)
}

fn check_boxed_number(boxed_num: u64) -> bool {
    return (boxed_num & FIXNUM_MASK) == FIXNUM_TAG;
}

fn read_boxed_number(boxed_num: u64) -> Result<u64> {
    Ok(boxed_num >> FIXNUM_SHIFT)
}

fn check_boxed_character(boxed_char: u64) -> bool {
    return (boxed_char & CHAR_MASK) == CHAR_TAG;
}

fn read_boxed_character(boxed_char: u64) -> Result<char> {
    let char_value: u32 = (boxed_char >> CHAR_SHIFT).try_into().or(Err(
        eyre! {"Character value too high: {}", boxed_char >> CHAR_SHIFT},
    ))?;

    char::from_u32(char_value).ok_or(eyre!("Invalid character value {}", char_value))
}

fn check_boxed_boolean(boxed_boolean: u64) -> bool {
    return (boxed_boolean & BOOLEAN_MASK) == BOOLEAN_TAG;
}

fn read_boxed_boolean(boxed_boolean: u64) -> Result<bool> {
    let boolean_val = (boxed_boolean >> BOOLEAN_SHIFT) & 1;

    return Ok(boolean_val == 1);
}

fn print_boxed(boxed: u64) -> Result<String> {
    if check_boxed_number(boxed) {
        return Ok(format!("{}", read_boxed_number(boxed)?));
    }

    if check_boxed_character(boxed) {
        return Ok(format!("#\\{}", read_boxed_character(boxed)?));
    }

    if check_boxed_boolean(boxed) {
        let b = read_boxed_boolean(boxed)?;
        if b {
            return Ok("#t".to_string());
        } else {
            return Ok("#f".to_string());
        }
    }

    Err(eyre!("Invalid tagged value {}", boxed))
}

fn interpret(code: Vec<u64>) -> Result<u64> {
    let mut pc: usize = 0;
    let mut stack: Vec<u64> = Vec::new();

    let mut readword = || -> Result<u64> {
        if pc >= code.len().try_into()? {
            return Err(eyre!("Program ends without return"));
        }

        let word = code[pc];
        pc += 1;
        Ok(word)
    };

    loop {
        let instr = read_op(readword()?)?;
        dbg!(instr);
        match instr {
            Op::LOAD64 => {
                let boxed_num = readword()?;
                if !check_boxed_number(boxed_num) {
                    return Err(eyre!("Invalid tagged number value {}", boxed_num));
                }

                stack.push(boxed_num)
            }

            Op::LOAD_CHAR => {
                let boxed_char = readword()?;
                if !check_boxed_character(boxed_char) {
                    return Err(eyre!("Invalid tagged character value {}", boxed_char));
                }

                stack.push(boxed_char)
            }

            Op::LOAD_BOOL => {
                let boxed_boolean = readword()?;
                if !check_boxed_boolean(boxed_boolean) {
                    return Err(eyre!("Invalid tagged boolean value {}", boxed_boolean));
                }

                stack.push(boxed_boolean)
            }

            Op::RETURN => {
                let val = stack.pop().ok_or(eyre!("No return value"));
                return val;
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

    dbg!(ret);
    let print_val = print_boxed(ret)?;

    println!("{}", print_val);

    Ok(())
}
