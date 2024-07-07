

use anyhow::bail;
use anyhow::Result;
use std::collections::VecDeque;
use std::iter::Peekable;
use std::path::PathBuf;

#[derive(Debug, PartialEq, Clone)]
pub(crate) enum Pattern {
    Dir(PathBuf, Box<Pattern>),
    File(PathBuf),
    Recurse(Box<Pattern>),
}

impl Pattern {
    pub fn parse(pat: &str) -> Result<Pattern> {
        Self::parse_(Lexer::new(pat).peekable())
    }

    fn parse_(mut tokens: Peekable<Lexer>) -> Result<Pattern> {
        match tokens.next() {
            Some(Token::Ellipsis) => Ok(Pattern::Recurse(Box::new(Self::parse_(tokens)?))),
            Some(Token::Elem(elem)) if tokens.peek().is_none() => Ok(Pattern::File(elem.into())),
            Some(Token::Elem(elem)) => {
                Ok(Pattern::Dir(elem.into(), Box::new(Self::parse_(tokens)?)))
            }
            _ => bail!("unexpected end of pattern"),
        }
    }
}

#[derive(Debug, PartialEq)]
enum Token<'a> {
    Elem(&'a str),
    Ellipsis,
}

struct Lexer<'a> {
    tokens: VecDeque<&'a str>,
}

impl<'a> Lexer<'a> {
    fn new(pattern: &str) -> Lexer {
        Lexer {
            tokens: chop(&pattern, &["/", "..."]),
        }
    }
}

impl<'a> Iterator for Lexer<'a> {
    type Item = Token<'a>;

    fn next(&mut self) -> Option<Token<'a>> {
        loop {
            match self.tokens.pop_front() {
                None => return None,
                Some("/") => continue,
                Some("...") => return Some(Token::Ellipsis),
                Some(elem) => return Some(Token::Elem(elem)),
            }
        }
    }
}

fn chop<'a, T>(mut s: &'a str, delims: &[&'static str]) -> T
where
    T: FromIterator<&'a str>,
{
    std::iter::from_fn(|| {
        if s.is_empty() {
            return None;
        }

        match delims
            .iter()
            .enumerate()
            .flat_map(|(index, d)| match s.find(d) {
                Some(pos) => Some((index, pos)),
                None => None,
            })
            .min_by_key(|&(_, v)| v)
        {
            Some((index, 0)) => {
                let delim = delims[index];
                s = &s[delim.len()..];
                Some(delim)
            }
            Some((_, pos)) => {
                let token = &s[..pos];
                s = &s[pos..];
                Some(token)
            }
            None => {
                let token = s;
                s = "";
                Some(token)
            }
        }
    })
    .collect()
}

// edit icsp...if/*.thrift
// edit cpplatform.../

// Literal(match), Recurse(...), Glob,
// MatchComponent

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn chop_basic() {
        assert_eq!(
            chop::<Vec<&str>>("foo/bar/baz..bar...biz/foo", &["/", "..."]),
            vec!["foo", "/", "bar", "/", "baz..bar", "...", "biz", "/", "foo",]
        );
    }

    #[test]
    fn chop_prefix() {
        assert_eq!(
            chop::<Vec<&str>>("...foo", &["/", "..."]),
            vec!["...", "foo"],
        )
    }

    #[test]
    fn basic_parse() {
        assert_eq!(
            Pattern::parse("hello...foo...bar/baz").unwrap(),
            Pattern::Dir(
                "hello".into(),
                Box::new(Pattern::Recurse(Box::new(Pattern::Dir(
                    "foo".into(),
                    Box::new(Pattern::Recurse(Box::new(Pattern::Dir(
                        "bar".into(),
                        Box::new(Pattern::File("baz".into()))
                    ))))
                ))))
            )
        )
    }

    #[test]
    fn parse_ellipsis() {
        assert_eq!(
            Pattern::parse("...bar.foo").unwrap(),
            Pattern::Recurse(Box::new(Pattern::File("bar.foo".into())))
        );
    }

    #[test]
    fn basic_lex() {
        assert_eq!(
            Lexer::new("x/y/z...f..oo/bar").collect::<Vec<Token>>(),
            vec![
                Token::Elem("x"),
                Token::Elem("y"),
                Token::Elem("z"),
                Token::Ellipsis,
                Token::Elem("f..oo"),
                Token::Elem("bar"),
            ],
        );
    }
}
