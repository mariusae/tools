
mod eval;
mod pattern;

use anyhow::Context;
use clap::{Arg, Command};
use std::env;
use std::path::Path;

use eval::Eval;
use pattern::Pattern;

fn main() {
    env_logger::init();
    let matches = Command::new("myapp")
        .version("1.0")
        .about("todo")
        .arg(
            Arg::new("pattern")
                .help("search pattern")
                .required(true)
                .index(1),
        )
        .get_matches();

    let pattern = Pattern::parse(matches.get_one::<String>("pattern").unwrap())
        .context("failed to parse pattern")
        .unwrap();

    let edit_path = env::var("EDITPATH").ok().unwrap_or(".".into());
    let paths: Vec<&str> = edit_path.split(":").filter(|s| !s.is_empty()).collect();

    for path in paths {
        let eval = Eval::new(Path::new(path), &pattern);

        for path in eval.eval_to_iter() {
            println!("{}", path.display());
        }
    }
}
