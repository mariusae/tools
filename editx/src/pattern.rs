

use anyhow::bail;
use anyhow::Result;
use std::collections::VecDeque;
use std::iter::Peekable;
use std::path::PathBuf;

#[derive(Debug, PartialEq, Clone)]
pub(crate) enum Component {
    Full(String),
    Prefix(String),
    Suffix(String),
    Everything,
    Nothing,
}

impl Component {
    pub fn matches(&self, s: &str) -> bool {
        match self {
            Component::Full(full) => s == full,
            Component::Prefix(prefix) => s.starts_with(prefix),
            Component::Suffix(suffix) => s.ends_with(suffix),
            Component::Everything => true,
            Component::Nothing => false,
        }
    }

    fn parse(component: &str) -> Component {
        if component.len() == 0 {
            return Component::Nothing;
        }

        let chars: Vec<char> = component.chars().collect();

        if chars[0] == '*' {
            if chars.len() == 1 {
                Component::Everything
            } else {
                Component::Suffix(chars[1..].iter().collect())
            }
        } else if chars[chars.len() - 1] == '*' {
            Component::Prefix(chars[..chars.len() - 1].iter().collect())
        } else {
            Component::Full(chars.iter().collect())
        }
    }
}

#[derive(Debug, PartialEq, Clone)]
pub(crate) enum Pattern {
    Dir(Component, Box<Pattern>),
    File(Component),
    Recurse(Box<Pattern>),
}

impl Pattern {
    pub fn parse(pat: &str) -> Result<Pattern> {
        Self::parse_pattern(Lexer::new(pat).peekable())
    }

    fn parse_pattern(mut tokens: Peekable<Lexer>) -> Result<Pattern> {
        match tokens.next() {
            Some(Token::Ellipsis) => Ok(Pattern::Recurse(Box::new(Self::parse_pattern(tokens)?))),
            Some(Token::Elem(elem)) if tokens.peek().is_none() => {
                Ok(Pattern::File(Component::parse(elem)))
            }
            Some(Token::Elem(elem)) => Ok(Pattern::Dir(
                Component::parse(elem),
                Box::new(Self::parse_pattern(tokens)?),
            )),
            _ => bail!("unexpected end of pattern"),
        }
    }
}

#[derive(Debug, PartialEq)]
enum Token<'a> {
    Elem(&'a str),
    Ellipsis,
    Star,
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
                Some("*") => return Some(Token::Star),
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
    fn test_component_parse() {
        assert_eq!(Component::parse("*bar"), Component::Suffix("bar".into()));
        assert_eq!(Component::parse("bar*"), Component::Prefix("bar".into()));
        assert_eq!(Component::parse("bar"), Component::Full("bar".into()));
    }

    #[test]
    fn test_component_match() {
        let component = Component::parse("*bar");
        assert!(component.matches("foobar"));
        assert!(component.matches("bar"));
        assert!(!component.matches("baz"));
        assert!(!component.matches("foobarb"));
    }

    #[test]
    fn basic_parse() {
        assert_eq!(
            Pattern::parse("hello...foo...bar/baz").unwrap(),
            Pattern::Dir(
                Component::Full("hello".into()),
                Box::new(Pattern::Recurse(Box::new(Pattern::Dir(
                    Component::Full("foo".into()),
                    Box::new(Pattern::Recurse(Box::new(Pattern::Dir(
                        Component::Full("bar".into()),
                        Box::new(Pattern::File(Component::Full("baz".into())))
                    ))))
                ))))
            )
        )
    }

    #[test]
    fn parse_ellipsis() {
        assert_eq!(
            Pattern::parse("...bar.foo").unwrap(),
            Pattern::Recurse(Box::new(Pattern::File(Component::Full("bar.foo".into()))))
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
