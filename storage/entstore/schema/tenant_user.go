package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type TenantUser struct{ ent.Schema }

func (TenantUser) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "tenant_user"},
	}
}

func (TenantUser) Fields() []ent.Field {
	return []ent.Field{
		field.Uint("id").Unique().Immutable(),
		field.String("user_id").MaxLen(100),
		field.String("tenant_id").MaxLen(100),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.Time("deleted_at").Optional().Nillable(),
	}
}

func (TenantUser) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "tenant_id").Unique().
			StorageKey("idx_tenant_user_unique"),
		index.Fields("deleted_at").StorageKey("idx_tenant_user_deleted_at"),
	}
}
