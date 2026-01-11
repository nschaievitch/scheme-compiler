use eyre::{Result, eyre};

#[repr(u64)]
pub enum Op {
    LOAD64,
    RETURN,
}

fn read_op(opcode: u64) -> Result<Op> {
    match opcode {
        LOAD64 => Ok(Op::LOAD64),
        RETURN => Ok(Op::RETURN),
        _ => Err(eyre!("Unsupported opcode: {}", opcode)),
    }
}

fn interpret(code: Vec<u64>) -> Result<u64> {
    let pc: u64 = 0;
    let mut stack: Vec<u64> = Vec::new();

    Ok(1)
}

fn main() {
    println!("Hello, world!");
}
