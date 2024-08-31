# k8x

## Acknowledgements

My experience with real world k8s apps is limited. I work professionally as a developer, so my perspective is biased. I draw a lot of impressions from coding and managing simple applications with docker and docker compose. I am using helm and k8s at work and some of my frustrations might come from inexperience or seem perfectly fine for experienced k8s admins or ops people.

## Features:

- .env integration
  - K8X_MY_VARIABLE
- Automatic namespace handling
- Chart sharing via npm
- Better templating
- Statically typed
- IDE support
    ![Proper intellisense support](assets/images/proper_intellisense_support.png "Proper intellisense support")
- Single File definition
- Multi cluster definition
- Reusable components
- Versioning
- Packaging

## Usage

```
k8x install my-wordpress
k8x install my-wordpress -u
k8x update my-wordpress
k8x compile
k8x ls
k8x rm
```

## Goals

- Single file app
- Inventory management
- Mature tooling
- Easy sharing
- Easy reusability
- Flexibility

## Non Goals

## Example chart

```tsx
const replicas = Number(process.env["VARIABLE"]);

export default () => (
  <cluster config="~/.kube/config">
    <namespace name="default">
      <deployment>
        <spec replicas={replicas}>
          <selector>
            <match-label key="appp">snowflake</match-label>
            <match-expression key="component" operator="In" values={["cache"]}>
              redis
            </match-expression>
          </selector>
        </spec>

        <template>
          <spec>
            <container
              image="registry.k8s.io/serve_hostname"
              imagePullPolicy="Always"
              name="snowflake"
            ></container>
          </spec>
        </template>
      </deployment>

      <service>
        <spec>
          <selector>
            <match-label key="app.kubernetes.io/name">MyApp</match-label>
          </selector>

          <port protocol="TCP" port={80} targetPort={80} />
        </spec>
      </service>
    </namespace>
  </cluster>
);
```

## Terminology

### Component

A piece of tsx code, that is typically a kubernetes entity, for example ingress, pod, service

### Tag

A tsx directive for example <cluster> or <namespace>

## FAQ

### Why JSX/TSX

JSX and TSX already are very mature tooling built directly into the tsc toolchain. IDE support is superior compared to simple yaml or other templating engines. Node has a very mature package and easy accessible code/chart sharing mechanism.
