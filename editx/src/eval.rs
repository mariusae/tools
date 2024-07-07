
use crate::pattern::Pattern;
use std::borrow::Cow;
use std::fs;
use std::io;
use std::iter;
use std::path::Path;
use std::path::PathBuf;
use walkdir::WalkDir;

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
            Pattern::Dir(path, ref next) => {
                let path = self.path.join(path);
                match fs::metadata(&path) {
                    Result::Ok(meta) if meta.is_dir() => Self {
                        path: Cow::Owned(path),
                        pattern: next,
                    }
                    .eval_to_iter(),
                    Result::Ok(_) => Box::new(iter::empty()),
                    Err(err) if err.kind() == io::ErrorKind::NotFound => Box::new(iter::empty()),
                    Err(err) => Box::new({
                        iter::from_fn(move || {
                            eprintln!("skipped {:?}: stat: {:?}", path, err);
                            None
                        })
                        .fuse()
                    }),
                }
            }
            Pattern::File(path) => {
                let path = self.path.join(path);
                match fs::metadata(&path) {
                    Result::Ok(meta) if meta.is_file() => Box::new(iter::once(path)),
                    Result::Ok(_) => Box::new(iter::empty()),
                    Err(err) if err.kind() == io::ErrorKind::NotFound => Box::new(iter::empty()),
                    Err(err) => Box::new(
                        iter::from_fn(move || {
                            eprintln!("skipped {:?}: stat: {:?}", path, err);
                            None
                        })
                        .fuse(),
                    ),
                }
            }
            Pattern::Recurse(next) => match &**next {
                Pattern::Recurse(_) => panic!("invalid pattern"),
                Pattern::File(file_name) => Box::new(
                    WalkDir::new(&self.path)
                        .into_iter()
                        .filter_map(move |e| match e {
                            Result::Err(err) => {
                                eprintln!("skipped while walking: stat: {:?}", err);
                                None
                            }
                            Result::Ok(e) if e.file_name() == file_name => Some(e.into_path()),
                            Result::Ok(_) => None,
                        }),
                ),
                Pattern::Dir(dir_name, next) => {
                    Box::new(WalkDir::new(&self.path).into_iter().flat_map(move |e| {
                        match e {
                            Result::Err(err) => {
                                eprintln!("skipped while walking: stat: {:?}", err);
                                Box::new(iter::empty())
                            }
                            Result::Ok(e)
                                if e.file_type().is_dir() && e.file_name() == dir_name =>
                            {
                                Eval {
                                    path: Cow::Owned(e.into_path()),
                                    pattern: next,
                                }
                                .eval_to_iter()
                            }
                            Result::Ok(_) => Box::new(iter::empty()),
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
