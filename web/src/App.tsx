import { Route, Switch } from "wouter";
import { Layout } from "./Layout";
import Dashboard from "./pages/Dashboard";
import Scanners from "./pages/Scanners";
import Findings from "./pages/Findings";
import Settings from "./pages/Settings";

function App() {
  return (
    <Layout>
      <Switch>
        <Route path="/" component={Dashboard} />
        <Route path="/scanners" component={Scanners} />
        <Route path="/findings" component={Findings} />
        <Route path="/settings" component={Settings} />
        <Route>404: Page Not Found</Route>
      </Switch>
    </Layout>
  );
}

export default App;
