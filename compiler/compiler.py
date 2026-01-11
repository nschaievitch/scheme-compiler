import enum, unittest, string, sys

FIXNUM_MASK = 0b11
FIXNUM_TAG = 0b00
FIXNUM_SHIFT = 2
              
CHAR_MASK = 0b11111111
CHAR_TAG = 0b00001111
CHAR_SHIFT = 8
              
BOOLEAN_MASK= 0b1111111
BOOLEAN_TAG = 0b0011111
BOOLEAN_SHIFT =7

class Parser:
    def __init__(self, source: str):
        self.source = source
        self.pos = 0
        self.length = len(source)

    def peek(self):
        if self.pos == self.length:
            return ''
        return self.source[self.pos]
        
    def skip_whitespace(self):
        while self.peek() in string.whitespace:
            self.pos += 1

    def parse_number(self):
        number_string = ""
        while self.peek().isdigit():
            number_string += self.peek()
            self.pos += 1
        return int(number_string)
            

    def parse(self) -> object:
        self.skip_whitespace()
        match self.peek():
            case '':
                raise EOFError("Unexpected end of input")
            case c if c.isdigit():
                return self.parse_number()
            case "#":
                self.pos += 1
                if self.peek() == '\\':
                    self.pos += 1
                    return self.peek()
                elif self.peek() == "t":
                    return True
                elif self.peek() == "f":
                    return False
                else:
                    raise SyntaxError(f"Unknown literal #{self.peek()}")
            case c:
                raise SyntaxError ("Invalid syntax found: {c}")

def scheme_parse(source: str) -> object:
    return Parser(source).parse()

class Compiler:
    def __init__(self):
        self.code = []
        self.max_locals_count = 0
        
    def box_fixnum(self, expr):
        if expr >= 2 ** 62:
            raise NotImplementedError(f"Compiler does not yet support integers larger than 2^62 - 1. Got {expr}")
        
        return expr << FIXNUM_SHIFT
        
    def box_character(self, expr):
        num = ord(expr)

        return (num << CHAR_SHIFT) | CHAR_TAG
    
    def box_boolean(self, expr):
        val = 1 if expr else 0

        return (val << BOOLEAN_SHIFT) | BOOLEAN_TAG

    # We don't have envs yet, so for now I allow not passing it (to get compile_function working with this signature)
    def compile(self, expr, env=None):
        emit = self.code.append
        match expr:
            case bool(_):
                emit(I.LOAD_BOOL)
                emit(self.box_boolean(expr))
            case int(_):
                emit(I.LOAD64)
                emit(self.box_fixnum(expr))
            case str(_):
                if len(expr) > 1:
                    raise NotImplementedError("Strings are not yet supported")
                emit(I.LOAD_CHAR)
                emit(self.box_character(expr))
            case _:
                raise NotImplementedError(expr)

    def compile_function(self, expr):
        self.compile(expr)
        self.code.append(I.RETURN)
        
    def write_to_stream(self, f):
        for op in self.code:
            f.write(op.to_bytes(8, "little"))
        

class I(enum.IntEnum):
    LOAD64 = enum.auto()
    LOAD_BOOL = enum.auto()
    LOAD_CHAR = enum.auto()
    RETURN = enum.auto()

# TODO: add proper tests
class ParseTests(unittest.TestCase):
    def _parse(self, source: str) -> object:
        return Parser(source).parse()

    def test_parse_fixnum(self):
        self.assertEqual(self._parse("42"), 42)

    def test_parse_fixnum_with_whitespace(self):
        self.assertEqual(self._parse("     42"), 42)

    def test_parse_true(self):
        self.assertEqual(self._parse("#t"), True)

    def test_parse_false(self):
        self.assertEqual(self._parse("#f"), False)

class CompileTests(unittest.TestCase):
    def _compile_func(self, source: str) -> list[int]:
        c = Compiler()
        parsed = scheme_parse(source)
        c.compile_function(parsed)
        return c.code

    def test_compiles_fixnum(self):
        self.assertEqual(self._compile_func("42"), [I.LOAD64, (42 << FIXNUM_SHIFT), I.RETURN])
    
    def test_compiles_boolean(self):
        self.assertEqual(self._compile_func("#t"), [I.LOAD_BOOL, (1 << BOOLEAN_SHIFT) | BOOLEAN_TAG, I.RETURN])

def compile_program():
    source = sys.stdin.read()
    program = scheme_parse(source)
    compiler = Compiler()
    compiler.compile_function(program)
    compiler.write_to_stream(sys.stdout.buffer)

if __name__ == '__main__':
    compile_program()