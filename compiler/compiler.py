import enum, unittest, string, sys

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
            case c:
                raise NotImplementedError(f"Parser only supports numbers currently. Found {c}")

def scheme_parse(source: str) -> object:
    return Parser(source).parse()

class Compiler:
    def __init__(self):
        self.code = []
        self.max_locals_count = 0
        
    def box_fixnum(self, expr):
        if expr >= 2 ** 62:
            raise NotImplementedError(f"Compiler does not yet support integers larger than 2^62 - 1. Got {expr}")
        
        # mask = 0b11
        # tag = 0b00
        
        return expr << 2
        

    # We don't have envs yet, so for now I allow not passing it (to get compile_function working with this signature)
    def compile(self, expr, env=None):
        emit = self.code.append
        match expr:
            case int(_):
                emit(I.LOAD64)
                emit(self.box_fixnum(expr))

    def compile_function(self, expr):
        self.compile(expr)
        self.code.append(I.RETURN)
        
    def write_to_stream(self, f):
        for op in self.code:
            f.write(op.to_bytes(8, "little"))
        

class I(enum.IntEnum):
    # Starts at 0 and increments
    LOAD64 = enum.auto()
    RETURN = enum.auto()

# TODO: add proper tests
class ParseTests(unittest.TestCase):
    def _parse(self, source: str) -> object:
        return Parser(source).parse()

    def test_parse_fixnum(self):
        self.assertEqual(self._parse("42"), 42)

    def test_parse_fixnum_with_whitespace(self):
        self.assertEqual(self._parse("     42"), 42)


class CompileTests(unittest.TestCase):
    def _compile_func(self, source: str) -> list[int]:
        c = Compiler()
        c.compile_function(source)
        return c.code

    def test_compiles_fixnum(self):
        self.assertEqual(self._compile_func(42), [I.LOAD64, (42 << 2), I.RETURN])

def compile_program():
    source = sys.stdin.read()
    program = scheme_parse(source)
    compiler = Compiler()
    compiler.compile_function(program)
    compiler.write_to_stream(sys.stdout.buffer)

if __name__ == '__main__':
    # TODO: implement proper CLI
    if len(sys.argv) >= 2 and sys.argv[1] == "test":
        unittest.main()
    else:
        compile_program()