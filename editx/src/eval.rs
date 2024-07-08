
use crate::pattern::Component;
use crate::pattern::Pattern;
use std::borrow::Cow;
use std::fs;
use std::io;
use std::iter;
use std::path::Path;
use std::path::PathBuf;
use walkdir::WalkDir;

macro_rules! error_iter {
    // This pattern allows for a format string with arguments
    ($fmt:expr, $($arg:tt)*) => {
        Box::new(iter::from_fn(move || {
            println!($fmt, $($arg)*);
            None
        }).fuse())
    };
}

#[derive(Debug)]
pub(crate) struct Eval<'a> {
    path: Cow<'a, Path>,
    pattern: &'a Pattern,
}

impl<'a> Eval<'a> {
    pub fn new(path: &'a Path, pattern: &'a Pattern) -> Eval<'a> {
        Eval {
            path: Cow::Borrowed(path),
            pattern,
        }
    }

    pub fn eval_to_iter(&self) -> Box<dyn Iterator<Item = PathBuf> + 'a> {
        log::debug!("eval_to_iter {:?}", self);
        match &self.pattern {
            Pattern::Dir(Component::Full(ref elem), ref next) => {
                let path = self.path.join(elem);
                match fs::metadata(&path) {
                    Ok(meta) if meta.is_dir() => Self {
                        path: Cow::Owned(path),
                        pattern: next,
                    }
                    .eval_to_iter(),
                    Ok(_) => Box::new(iter::empty()),
                    Err(err) if err.kind() == io::ErrorKind::NotFound => Box::new(iter::empty()),
                    Err(err) => error_iter!("skipped {:?}: stat: {:?}", path, err),
                }
            }
            Pattern::Dir(ref glob, ref next) => {
                let cloned_path = self.path.clone();
                let cloned_glob = glob.clone();
                match fs::read_dir(&self.path.clone()) {
                    Err(err) => error_iter!("skipped {:?}: stat: {:?}", cloned_path, err),
                    // TODO: check is file
                    Ok(read_dir) => Box::new(read_dir.flat_map(move |entry| {
                        match entry {
                            Err(err) => {
                                eprintln!("read_dir {:?}: skipped entry: {:?}", cloned_path, err);
                                Box::new(iter::empty())
                            }
                            // TODO: check is dir
                            Ok(entry) if cloned_glob.matches(entry.path().to_str().unwrap()) => {
                                Self {
                                    path: Cow::Owned(entry.path()),
                                    pattern: next,
                                }
                                .eval_to_iter()
                            }
                            Ok(_) => Box::new(iter::empty()),
                        }
                    })),
                }
            }
            Pattern::File(Component::Full(ref elem)) => {
                let path = self.path.join(elem);
                match fs::metadata(&path) {
                    Ok(meta) if meta.is_file() => Box::new(iter::once(path)),
                    Ok(_) => Box::new(iter::empty()),
                    Err(err) if err.kind() == io::ErrorKind::NotFound => Box::new(iter::empty()),
                    Err(err) => error_iter!("skipped {:?}: stat: {:?}", path, err),
                }
            }
            Pattern::File(ref glob) => {
                let cloned_path = self.path.clone();
                let cloned_glob = glob.clone();
                match fs::read_dir(&self.path.clone()) {
                    Err(err) => error_iter!("skipped {:?}: stat: {:?}", cloned_path, err),
                    // TODO: check is file
                    Ok(read_dir) => Box::new(read_dir.flat_map(move |entry| match entry {
                        Err(err) => {
                            eprintln!("read_dir {:?}: skipped entry: {:?}", cloned_path, err);
                            None
                        }
                        Ok(entry) if cloned_glob.matches(entry.path().to_str().unwrap()) => {
                            Some(entry.path())
                        }
                        Ok(_) => None,
                    })),
                }
            }
            Pattern::Recurse(next) => match &**next {
                Pattern::Recurse(_) => panic!("invalid pattern"),
                Pattern::File(component) => Box::new(
                    WalkDir::new(&self.path)
                        .into_iter()
                        .filter_map(move |e| match e {
                            Err(err) => {
                                eprintln!("skipped while walking: stat: {:?}", err);
                                None
                            }
                            Ok(e) if component.matches(e.file_name().to_str().unwrap()) => {
                                Some(e.into_path())
                            }
                            Ok(_) => None,
                        }),
                ),
                Pattern::Dir(component, next) => {
                    Box::new(WalkDir::new(&self.path).into_iter().flat_map(move |e| {
                        match e {
                            Err(err) => {
                                eprintln!("skipped while walking: stat: {:?}", err);
                                Box::new(iter::empty())
                            }
                            Ok(e)
                                if e.file_type().is_dir()
                                    && component.matches(e.file_name().to_str().unwrap()) =>
                            {
                                Eval {
                                    path: Cow::Owned(e.into_path()),
                                    pattern: next,
                                }
                                .eval_to_iter()
                            }
                            Ok(_) => Box::new(iter::empty()),
                        }
                    }))
                }
            },
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    use std::path::Path;

    #[test]
    fn test_path() {
        let path = Path::new("x");

        let base = Path::new("blah");
        let path = base.join(path);
    }
}
