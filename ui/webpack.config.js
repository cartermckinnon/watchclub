const path = require("path");
const HtmlWebpackPlugin = require("html-webpack-plugin");

module.exports = {
  mode: "development",
  entry: {
    index: "./src/index.js",
  },
  devtool: "inline-source-map",
  devServer: {
    port: 3000,
    static: [
      {
        directory: path.join(__dirname, "build"),
      },
      {
        directory: path.join(__dirname, "src/css"),
        publicPath: "/css",
      },
    ],
    proxy: [
      {
        context: ['/watchclub.WatchClubService'],
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    ],
  },
  plugins: [
    new HtmlWebpackPlugin({
      template: "src/index.html",
      inject: 'body',
    })
  ],
  output: {
    filename: "[name].bundle.js",
    path: path.resolve(__dirname, "build"),
    clean: true,
  }
};