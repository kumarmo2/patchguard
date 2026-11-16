use std::{env, error::Error, net::SocketAddr};

use axum::{Router, routing::post};
use tokio::net::TcpListener;

#[tokio::main]
async fn main() -> Result<(), Box<dyn Error>> {
    let port = match env::var("PORT") {
        Ok(value) => value.parse::<u16>()?,
        Err(env::VarError::NotPresent) => 8888,
        Err(error) => return Err(error.into()),
    };
    let address = SocketAddr::from(([127, 0, 0, 1], port));
    let listener = TcpListener::bind(address).await?;
    let app = Router::new().route("/webhooks/github", post(|| async { "OK" }));

    println!("review-bot-api listening on {}", listener.local_addr()?);
    axum::serve(listener, app).await?;
    Ok(())
}
